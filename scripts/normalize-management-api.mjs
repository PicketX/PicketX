// SPDX-License-Identifier: Apache-2.0
// Deterministic corrections for selected Swaggo rc6 gaps, checked against handlers.
import fs from "node:fs";
const path = "docs/api/swagger.json";
const spec = JSON.parse(fs.readFileSync(path, "utf8"));
spec.servers = [{url: "/"}];
const request = spec.components.schemas["controller.PublicationRequest"];
request.additionalProperties = false;
for (const property of Object.values(request.properties)) property.minLength = 1;
fs.writeFileSync(path, JSON.stringify(spec, null, 4) + "\n");
