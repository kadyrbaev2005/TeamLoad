import { describe, expect, it } from "vitest";
import { nextRaci, raciCheck } from "./raci";

describe("RACI", () => {
  it("cycles empty → R → A → C → I → empty", () => {
    expect(nextRaci("")).toBe("R");
    expect(nextRaci("R")).toBe("A");
    expect(nextRaci("A")).toBe("C");
    expect(nextRaci("C")).toBe("I");
    expect(nextRaci("I")).toBe("");
  });
  it("checks one A and at least one R", () => {
    const ids = ["s1", "s2", "s3"];
    const row = (cells: Record<string, "" | "R" | "A" | "C" | "I">) => ({ id: "a", name: "x", cells });
    expect(raciCheck(row({ s1: "A", s2: "R" }), ids)).toBe("");
    expect(raciCheck(row({ s1: "R", s2: "R" }), ids)).toBe("No A");
    expect(raciCheck(row({ s1: "A", s2: "A", s3: "R" }), ids)).toBe("Several A");
    expect(raciCheck(row({ s1: "A", s2: "C" }), ids)).toBe("No R");
    // Values of seats outside the plan are ignored.
    expect(raciCheck(row({ s1: "A", s2: "R", old: "A" }), ids)).toBe("");
  });
});
