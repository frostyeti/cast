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
const output = Deno.env.get("INPUT_OUTPUT") || Deno.env.get("OUTPUT");
const runtime = Deno.env.get("INPUT_RUNTIME") || Deno.env.get("RUNTIME");

const args = ["publish", project, "-c", configuration];

if (!restore) {
    args.push("--no-restore");
}

if (noBuild) {
    args.push("--no-build");
}

if (output) {
    args.push("-o", output);
}

if (runtime) {
    args.push("-r", runtime);
}

console.log(`Running: dotnet ${args.join(" ")}`);
await run(["dotnet", ...args]);
