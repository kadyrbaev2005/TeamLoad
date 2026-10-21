import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Member } from "../api/types";
import { LoadChart } from "./LoadChart";
import { ConfirmModal } from "./Modal";
import { TaskModal } from "./TaskModal";

const member: Member = {
  id: "u1", name: "Dana Bekova", first_name: "Dana", initials: "DB", color: "--r5", is_owner: false, has_profile: true,
  seat_id: null, seat_name: null, role_confirmed: false, joined_at: "2026-10-22T00:00:00Z",
};

describe("TaskModal", () => {
  it("shows the exact validation messages and saves a valid task", async () => {
    const onSave = vi.fn();
    render(<TaskModal task={null} members={[member]} activities={[]} onSave={onSave} onClose={() => {}} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Add task" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Give the task a title.");
    await user.type(screen.getByLabelText("Title"), "Booking API");
    const est = screen.getByLabelText("Estimate (hours)");
    await user.clear(est);
    await user.click(screen.getByRole("button", { name: "Add task" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Enter an estimate greater than 0 hours");
    await user.type(est, "41");
    await user.click(screen.getByRole("button", { name: "Add task" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Estimate must be between 0.5 and 40 hours.");
    await user.clear(est);
    await user.type(est, "6");
    await user.selectOptions(screen.getByLabelText("Assignee"), "u1");
    await user.selectOptions(screen.getByLabelText("Status"), "doing");
    await user.click(screen.getByRole("button", { name: "Add task" }));
    expect(onSave).toHaveBeenCalledWith({ title: "Booking API", assignee_id: "u1", estimate: 6, status: "doing", raci_activity_id: null });
  });

  it("closes on Escape", () => {
    const onClose = vi.fn();
    render(<TaskModal task={null} members={[]} activities={[]} onSave={() => {}} onClose={onClose} />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});

describe("ConfirmModal", () => {
  it("is a labelled dialog that focuses the confirm button", () => {
    render(<ConfirmModal title="Regenerate?" message="This replaces the roles." confirmLabel="Regenerate" onConfirm={() => {}} onCancel={() => {}} />);
    expect(screen.getByRole("dialog", { name: "Regenerate?" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Regenerate" })).toHaveFocus();
  });
});

describe("LoadChart", () => {
  it("draws one bar per member, the band and the dashed average", () => {
    const { container } = render(
      <LoadChart
        average={15.2}
        rows={[
          { name: "Akzhol", active: 14, status: "Balanced" },
          { name: "Beknur", active: 22, status: "Overloaded" },
          { name: "Zhanarys", active: 9, status: "Underloaded" },
        ]}
      />,
    );
    const bars = container.querySelectorAll("rect[rx='5']");
    expect(bars).toHaveLength(3);
    expect(bars[1]?.getAttribute("fill")).toBe("var(--bad)");
    expect(bars[2]?.getAttribute("fill")).toBe("var(--warn)");
    expect(container.querySelector("line[stroke-dasharray='5 4']")).not.toBeNull();
    expect(screen.getByText("22 h")).toBeInTheDocument();
  });
});
