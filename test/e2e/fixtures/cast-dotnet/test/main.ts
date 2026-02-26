import { CI } from "jsr:@frostyeti/ci-env@0.0.0-alpha.3";
import { run } from "jsr:@frostyeti/exec@0.0.0-alpha.0.1.0";

const ci = CI || Deno.env.get("CI") === "true";
console.log("CI Detected:", ci);

const configuration = Deno.env.get("INPUT_CONFIGURATION") || Deno.env.get("CONFIGURATION") || (ci ? "Release" : "Debug");

// Determine restore flag
let restore = false;
const noRestoreInput = Deno.env.get("INPUT_NO_RESTORE");
if (noRestoreInput !== undefined && noRestoreInput !== "") {
    restore = noRestoreInput !== "true"; // if true, don't restore
} else {
    restore = !ci; // default to restoring if not in CI
}

const noBuildInput = Deno.env.get("INPUT_NO_BUILD");
const noBuild = noBuildInput === "true";

const project = Deno.env.get("INPUT_PROJECT") || Deno.env.get("PROJECT") || ".";
const filter = Deno.env.get("INPUT_FILTER") || Deno.env.get("FILTER");
const logger = Deno.env.get("INPUT_LOGGER") || Deno.env.get("LOGGER");

const args = ["test", project, "-c", configuration];

if (!restore) {
    args.push("--no-restore");
}

if (noBuild) {
    args.push("--no-build");
}

if (filter) {
    args.push("--filter", filter);
}

if (logger) {
    args.push("--logger", logger);
}

console.log(`Running: dotnet ${args.join(" ")}`);
await run(["dotnet", ...args]);
