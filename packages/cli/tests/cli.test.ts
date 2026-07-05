import { describe, it, expect } from "vitest";
import { parseArgs } from "../src/index.js";

describe("CLI", () => {
  it("should have required command exports in source", () => {
    const fs = require("fs");
    const source = fs.readFileSync("./src/index.ts", "utf-8");
    expect(source).toContain("export async function main()");
    expect(source).toContain("diff");
    expect(source).toContain("assert");
    expect(source).toContain("list-traces");
    expect(source).toContain("diff-batch");
  });

  it("should parse CLI arguments correctly", () => {
    const args = parseArgs(["node", "script", "diff", "--output", "json"]);
    expect(args._[0]).toBe("diff");
    expect(args.output).toBe("json");
  });

  it("should handle min-similarity argument", () => {
    const args = parseArgs(["node", "script", "assert", "--min-similarity", "85"]);
    expect(args["min-similarity"]).toBe(85);
  });
});
