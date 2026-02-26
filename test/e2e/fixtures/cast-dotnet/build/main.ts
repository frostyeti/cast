import { CI } from "jsr:@frostyeti/ci-env@0.0.0-alpha.3";
import { cmd, run } from "jsr:@frostyeti/exec@0.0.0-alpha.0.1.0";

const ci = CI || Deno.env.get("CI") === "true";
console.log("CI Environment Var:", Deno.env.get("CI"));
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

const project = Deno.env.get("INPUT_PROJECT") || Deno.env.get("PROJECT") || ".";

const args = ["build", project, "-c", configuration];

if (!restore) {
    args.push("--no-restore");
}

console.log(`Running: dotnet ${args.join(" ")}`);
await run(["dotnet", ...args]);
