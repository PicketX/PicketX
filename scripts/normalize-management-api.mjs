// SPDX-License-Identifier: Apache-2.0
// Deterministic corrections for selected Swaggo rc6 gaps, checked against handlers.
import fs from "node:fs";
const path = "docs/api/swagger.json";
const spec = JSON.parse(fs.readFileSync(path, "utf8"));
spec.servers = [{url: "/"}];
// An unreported Agent is represented by explicit null, never an empty assertion.
spec.components.schemas["agentsync.NodeObservation"].properties.report = {
  oneOf: [{ $ref: "#/components/schemas/agentsync.ReportedState" }, { type: "null" }],
};
const request = spec.components.schemas["controller.PublicationRequest"];
for (const name of ["controller.CreateAccessRequest", "controller.DecideAccessRequest"]) {
  const s = spec.components.schemas[name]; s.additionalProperties = false;
  if (s.properties.endpoint_names) { s.properties.endpoint_names.uniqueItems = true; Object.assign(s.properties.endpoint_names.items, { minLength: 1, maxLength: 1024 }); }
}
// Both transport aliases use the same handlers, but not the same credentials.
for (const [path, methods] of Object.entries(spec.paths)) if (path.startsWith('/console/api/v1/access-request')) {
  for (const [verb, operation] of Object.entries(methods)) {
    operation.security = [{ BrowserCookie: [] }]; operation.operationId = `browser_${operation.operationId}_${verb}`;
    if (verb === 'post') operation.description = `${operation.description ?? ''} Requires exact configured Origin and native Echo CSRF; legacy clients send X-CSRF-Token from session bootstrap. Strict JSON16KiB.`;
  }
}
request.additionalProperties = false;
for (const property of Object.values(request.properties)) property.minLength = 1;
const login = spec.components.schemas["controller.LoginRequest"];
const humanLogin = spec.components.schemas["controller.PasswordLoginRequest"];
humanLogin.additionalProperties = false;
// JSON Schema counts Unicode characters; the handler additionally checks the
// stricter UTF-8 byte bounds. Do not misdocument a 12-byte minimum as12characters.
Object.assign(humanLogin.properties.password, { writeOnly: true, minLength: 1, maxLength: 72, description: "12–72 UTF-8 bytes; no trimming" });
const bindingEdit = spec.components.schemas["controller.UpdateBindingEndpointsRequest"];
bindingEdit.additionalProperties = false;
bindingEdit.properties.reason.minLength = 1;
bindingEdit.properties.acknowledge_scope_change = { type: "boolean", const: true };
bindingEdit.properties.endpoint_names.uniqueItems = true;
Object.assign(bindingEdit.properties.endpoint_names.items, { minLength: 1, maxLength: 1024 });
const networkEdit = spec.components.schemas["controller.UpdateResourceNetworkRequest"];
networkEdit.additionalProperties = false;
networkEdit.properties.reason.minLength = 1;
for (const name of ["controller.IssueGrantRequest", "controller.RevokeGrantRequest", "controller.CreateResourceRequest", "controller.ResourceEndpointRequest", "controller.CreateBindingRequest", "controller.CreateHTTPRouteRequest", "controller.RemoveBindingRequest", "controller.RemoveResourceRequest", "controller.RemoveHTTPRouteRequest"]) {
  const grant = spec.components.schemas[name];
  grant.additionalProperties = false;
  for (const field of Object.values(grant.properties)) if (field.type === "string") field.minLength = 1;
}
spec.components.schemas["controller.RemoveBindingRequest"].properties.acknowledge_unprotection = { type: "boolean", const: true };
spec.components.schemas["controller.RemoveResourceRequest"].properties.acknowledge_removal = { type: "boolean", const: true };
spec.components.schemas["controller.RemoveHTTPRouteRequest"].properties.acknowledge_removal = { type: "boolean", const: true };
for (const name of ["controller.IssueGrantRequest", "controller.CreateBindingRequest"]) {
  const names = spec.components.schemas[name].properties.endpoint_names;
  names.uniqueItems = true;
  Object.assign(names.items, { minLength: 1, maxLength: 1024 });
}
const endpoint = spec.components.schemas["controller.ResourceEndpointRequest"].properties;
const route = spec.components.schemas["controller.CreateHTTPRouteRequest"].properties;
Object.assign(route.methods, { minItems: 1, maxItems: 7, uniqueItems: true });
route.methods.items = { type: "string", enum: ["GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"] };
Object.assign(route.paths, { minItems: 1, maxItems: 128, uniqueItems: true });
Object.assign(route.paths.items, { minLength: 1, maxLength: 8192 });
// rc6 promotes dive,required from each address to the optional parent list.
const endpointSchema = spec.components.schemas["controller.ResourceEndpointRequest"];
endpointSchema.required = endpointSchema.required.filter(name => name !== "addresses");
Object.assign(endpoint.port, { minimum: 1, maximum: 65535 });
Object.assign(endpoint.addresses, { minItems: 1, maxItems: 64 });
Object.assign(endpoint.addresses.items, { minLength: 1, maxLength: 128 });
login.additionalProperties = false;
login.properties.token.minLength = 32;
const decision = spec.components.schemas["controller.HTTPDecisionRequest"];
decision.additionalProperties = false;
for (const property of Object.values(decision.properties)) property.minLength = 1;
fs.writeFileSync(path, JSON.stringify(spec, null, 4) + "\n");
