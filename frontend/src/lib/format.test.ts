import { describe, expect, it } from "vitest";
import { addDaysISO, daysLeft, fmtDate, fmtH, shortName } from "./format";

describe("fmtH", () => {
  it("keeps one decimal and drops a trailing .0", () => {
    expect(fmtH(15.2)).toBe("15.2");
    expect(fmtH(14)).toBe("14");
    expect(fmtH(19.76)).toBe("19.8");
    expect(fmtH(3.0)).toBe("3");
  });
});

describe("dates", () => {
  it("formats en-GB like the design", () => {
    expect(fmtDate("2026-12-11")).toBe("11 Dec 2026");
  });
  it("counts days left and never goes negative", () => {
    const now = new Date("2026-10-22T09:00:00");
    expect(daysLeft("2026-12-11", now)).toBe(50);
    expect(daysLeft("2026-10-01", now)).toBe(0);
  });
  it("adds days", () => {
    expect(addDaysISO("2026-12-30", 7)).toBe("2027-01-06");
  });
});

describe("shortName", () => {
  it("abbreviates the surname", () => {
    expect(shortName("Aruzhan Sadykova")).toBe("Aruzhan S.");
    expect(shortName("Dana")).toBe("Dana");
  });
});
