// SPDX-License-Identifier: Apache-2.0
import fs from "node:fs";
import SwaggerParser from "@apidevtools/swagger-parser";
const spec = JSON.parse(fs.readFileSync(new URL("../../docs/api/swagger.json", import.meta.url), "utf8"));
// No URL/path resolver is enabled. Only the already-loaded local document and
// its internal references may be examined; validation has no remote input.
await SwaggerParser.validate(spec, {resolve: {external: false, file: false, http: false}});
if (spec.openapi !== "3.1.0") throw new Error("expected OpenAPI 3.1.0");
console.log("OpenAPI 3.1 schema/reference validation passed (local document only)");
