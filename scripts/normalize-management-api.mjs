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
request.additionalProperties = false;
for (const property of Object.values(request.properties)) property.minLength = 1;
const login = spec.components.schemas["controller.LoginRequest"];
for (const name of ["controller.IssueGrantRequest", "controller.RevokeGrantRequest", "controller.CreateResourceRequest", "controller.ResourceEndpointRequest", "controller.CreateBindingRequest", "controller.CreateHTTPRouteRequest"]) {
  const grant = spec.components.schemas[name];
  grant.additionalProperties = false;
  for (const field of Object.values(grant.properties)) if (field.type === "string") field.minLength = 1;
}
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
