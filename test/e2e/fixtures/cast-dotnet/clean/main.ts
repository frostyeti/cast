import { CI } from "jsr:@frostyeti/ci-env@0.0.0-alpha.3";
import { run } from "jsr:@frostyeti/exec@0.0.0-alpha.0.1.0";

const ci = CI || Deno.env.get("CI") === "true";
console.log("CI Detected:", ci);

const configuration = Deno.env.get("INPUT_CONFIGURATION") || Deno.env.get("CONFIGURATION") || (ci ? "Release" : "Debug");
const project = Deno.env.get("INPUT_PROJECT") || Deno.env.get("PROJECT") || ".";
const output = Deno.env.get("INPUT_OUTPUT") || Deno.env.get("OUTPUT");

const args = ["clean", project, "-c", configuration];

if (output) {
    args.push("-o", output);
}

console.log(`Running: dotnet ${args.join(" ")}`);
await run(["dotnet", ...args]);
