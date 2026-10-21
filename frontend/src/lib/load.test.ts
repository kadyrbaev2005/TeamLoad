import { describe, expect, it } from "vitest";
import { BAND_LABEL, loadStatus } from "./load";

describe("load status (shared thresholds)", () => {
  it("passes the reference case", () => {
    const loads = { Akzhol: 14, Beknur: 22, Yerasyl: 16, Irisbek: 15, Zhanarys: 9 };
    const avg = Object.values(loads).reduce((a, b) => a + b, 0) / 5;
    expect(avg).toBeCloseTo(15.2);
    expect(loadStatus(loads.Beknur, avg)).toBe("Overloaded");
    expect(loadStatus(loads.Zhanarys, avg)).toBe("Underloaded");
    expect(loadStatus(loads.Akzhol, avg)).toBe("Balanced");
    expect(loadStatus(loads.Yerasyl, avg)).toBe("Balanced");
    expect(loadStatus(loads.Irisbek, avg)).toBe("Balanced");
  });
  it("treats exactly 130% as balanced", () => {
    expect(loadStatus(13, 10)).toBe("Balanced");
    expect(loadStatus(13.1, 10)).toBe("Overloaded");
    expect(loadStatus(7.5, 10)).toBe("Balanced");
    expect(loadStatus(0, 0)).toBe("Balanced");
  });
  it("labels the band from the constants file", () => {
    expect(BAND_LABEL).toBe("75–130%");
  });
});
