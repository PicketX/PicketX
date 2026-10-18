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
    assert db.execute('SELECT version FROM controller_schema').fetchone()==(1,)
    db.close()
    p.send_signal(signal.SIGTERM);assert p.wait(timeout=7)==0
    print('static SQLite Controller: health/readiness, persisted revision, gRPC and SIGTERM passed')
finally:
    if p.poll() is None: p.kill();p.wait(timeout=3)
PY
