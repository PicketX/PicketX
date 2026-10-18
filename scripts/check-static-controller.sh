#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags 'netgo osusergo sqlite_omit_load_extension' -ldflags '-linkmode external -extldflags -static' -o "$work/picketx-controller" ./cmd/picketx-controller
file "$work/picketx-controller"
if readelf -l "$work/picketx-controller" | grep -q INTERP; then
  echo 'Controller artifact unexpectedly needs an ELF interpreter' >&2
  exit 1
fi
python3 - "$work" <<'PY'
import hashlib,json,os,pathlib,secrets,signal,sqlite3,subprocess,sys,time,select,urllib.request
work=pathlib.Path(sys.argv[1]);binary=work/'picketx-controller'
token=secrets.token_hex(32);tokenfile=work/'token';tokenfile.write_text(token);tokenfile.chmod(0o600)
creds=work/'credentials.json';creds.write_text(json.dumps({'credentials':[{'id':'smoke','agent_ref':'agent-a','sha256':hashlib.sha256(token.encode()).hexdigest()}]}));creds.chmod(0o600)
config=work/'controller.yaml';config.write_text('http: {listen: "127.0.0.1:0"}\nagent: {enabled: true, authority: "static:example-v1", listen: "127.0.0.1:0"}\nsource: {policy: "examples/static/gitea.yaml", facts: "examples/static/facts.json"}\ndatabase: '+json.dumps(str(work/'state'/'controller.sqlite'))+'\ncredentials: '+json.dumps(str(creds))+'\n')
p=subprocess.Popen([str(binary),'-config',str(config)],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
try:
    pending=b'';addresses=None;until=time.monotonic()+15
    while time.monotonic()<until and addresses is None:
        ready,_,_=select.select([p.stderr],[],[],0.2)
        if ready:
            chunk=os.read(p.stderr.fileno(),65536)
            if not chunk: raise RuntimeError('artifact exited before startup')
            pending+=chunk
            while b'\n' in pending:
                line,pending=pending.split(b'\n',1);event=json.loads(line)
                if event.get('msg')=='controller started': addresses=event
        if p.poll() is not None: raise RuntimeError('artifact startup failed')
    if addresses is None: raise RuntimeError('artifact startup deadline exceeded')
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
    for route in ('healthz','readyz'):
        with opener.open('http://'+addresses['http_listen']+'/'+route,timeout=2) as r:
            assert r.status==200
    subprocess.run(['go','run','./scripts/verify-controller',addresses['agent_listen'],str(tokenfile)],check=True,timeout=30)
    db=sqlite3.connect(work/'state'/'controller.sqlite')
    assert db.execute('SELECT authority,revision FROM controller_publication').fetchone()==('static:example-v1',1)
    assert db.execute('SELECT version FROM controller_schema').fetchone()==(2,)
    assert db.execute('SELECT mode FROM controller_source').fetchone()==('file',)
    db.close()
    p.send_signal(signal.SIGTERM);assert p.wait(timeout=7)==0
    print('static SQLite Controller: health/readiness, persisted revision, gRPC and SIGTERM passed')
finally:
    if p.poll() is None: p.kill();p.wait(timeout=3)

# Same actual fully-static artifact, explicit database authority and independent API principal.
admin=secrets.token_hex(32);principals=work/'principals.json'
principals.write_text(json.dumps({'principals':[{'id':'principal:smoke','sha256':hashlib.sha256(admin.encode()).hexdigest(),'roles':['role:admin']}]}));principals.chmod(0o600)
managed_db=work/'managed'/'controller.sqlite'
config.write_text('http: {listen: "127.0.0.1:0"}\nagent: {enabled: true, authority: "static:example-v1", listen: "127.0.0.1:0"}\nsource: {mode: database}\nmanagement: {credentials: '+json.dumps(str(principals))+'}\ndatabase: '+json.dumps(str(managed_db))+'\ncredentials: '+json.dumps(str(creds))+'\n')
def launch_managed():
    proc=subprocess.Popen([str(binary),'-config',str(config)],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
    try:
        pending=b'';until=time.monotonic()+15
        while time.monotonic()<until:
            readable,_,_=select.select([proc.stderr],[],[],0.2)
            if readable:
                chunk=os.read(proc.stderr.fileno(),65536)
                if not chunk: raise RuntimeError('managed artifact exited before startup')
                pending+=chunk
                while b'\n' in pending:
                    line,pending=pending.split(b'\n',1);event=json.loads(line)
                    if event.get('msg')=='controller started': return proc,event
            if proc.poll() is not None: raise RuntimeError('managed artifact startup failed')
        raise RuntimeError('managed artifact startup deadline exceeded')
    except BaseException:
        if proc.poll() is None: proc.kill();proc.wait(timeout=3)
        raise
def call_api(address,method,body=None,match=None):
    headers={'Authorization':'Bearer '+admin,'Content-Type':'application/json'}
    if match is not None: headers['If-Match']=match
    request=urllib.request.Request('http://'+address+'/api/v1/publication',data=body,headers=headers,method=method)
    with opener.open(request,timeout=3) as response:
        assert response.status==200
        return response.headers,json.load(response)
for boot in range(2):
    proc,addresses=launch_managed()
    try:
        address=addresses['http_listen'];headers,document=call_api(address,'GET')
        if boot==0:
            assert document['revision']==1 and headers['ETag']=='"1"'
            body=json.dumps({'policy':pathlib.Path('examples/static/gitea.yaml').read_text(),'facts':pathlib.Path('examples/static/facts.json').read_text(),'reason':'static managed smoke'}).encode()
            headers,document=call_api(address,'PUT',body,'"1"')
        assert document['revision']==2 and headers['ETag']=='"2"'
        assert document['policy']==pathlib.Path('examples/static/gitea.yaml').read_text()
        db=sqlite3.connect(managed_db)
        assert db.execute('SELECT mode FROM controller_source').fetchone()==('database',)
        assert db.execute('SELECT actor,revision FROM controller_audit ORDER BY id').fetchall()==[('system:bootstrap',1),('principal:smoke',2)]
        db.close()
        proc.send_signal(signal.SIGTERM);assert proc.wait(timeout=7)==0
    finally:
        if proc.poll() is None: proc.kill();proc.wait(timeout=3)
print('static managed Controller: independent Token/Casbin, HTTP CAS, SQLite/audit, restart and SIGTERM passed')
PY
