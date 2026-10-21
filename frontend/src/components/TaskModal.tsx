import { useState } from "react";
import type { Member, RaciActivity, Task, TaskInput, TaskStatus } from "../api/types";
import { checkEstimate, MSG } from "../lib/validate";
import { Modal } from "./Modal";

export const STATUS_LABEL: Record<TaskStatus, string> = { todo: "To do", doing: "Doing", done: "Done" };

type Props = {
  task: Task | null;
  members: Member[];
  activities: RaciActivity[];
  defaultAssignee?: string | null;
  serverError?: string;
  busy?: boolean;
  onSave: (input: TaskInput) => void;
  onDelete?: () => void;
  onClose: () => void;
};

/** New / edit task modal from the design; its Status field is the non-drag fallback. */
export function TaskModal({ task, members, activities, defaultAssignee, serverError, busy, onSave, onDelete, onClose }: Props) {
  const isNew = !task;
  const [title, setTitle] = useState(task?.title ?? "");
  const [who, setWho] = useState(task ? (task.assignee_id ?? "") : (defaultAssignee ?? ""));
  const [hours, setHours] = useState(String(task?.estimate ?? 2));
  const [status, setStatus] = useState<TaskStatus>(task?.status ?? "todo");
  const [act, setAct] = useState(task?.raci_activity_id ?? "");
  const [err, setErr] = useState("");

  const submit = () => {
    if (!title.trim()) return setErr(MSG.taskTitle);
    const [h, e] = checkEstimate(hours);
    if (e) return setErr(e);
    setErr("");
    onSave({ title: title.trim(), assignee_id: who || null, estimate: h, status, raci_activity_id: act || null });
  };

  const shown = err || serverError;
  return (
    <Modal title={isNew ? "New task" : "Edit task"} onClose={onClose} onSubmit={submit}>
      <div className="field">
        <label htmlFor="tTitle">Title</label>
        <input className="input" id="tTitle" value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} required data-autofocus />
      </div>
      <div className="grid2">
        <div className="field">
          <label htmlFor="tWho">Assignee</label>
          <select className="select" id="tWho" value={who} onChange={(e) => setWho(e.target.value)}>
            <option value="">Unassigned</option>
            {members.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="tH">Estimate (hours)</label>
          <input className="input mono" id="tH" type="number" min={0.5} max={40} step={0.5} value={hours} onChange={(e) => setHours(e.target.value)} />
        </div>
      </div>
      <div className="field">
        <label htmlFor="tSt">Status</label>
        <select className="select" id="tSt" value={status} onChange={(e) => setStatus(e.target.value as TaskStatus)}>
          {(Object.keys(STATUS_LABEL) as TaskStatus[]).map((k) => (
            <option key={k} value={k}>
              {STATUS_LABEL[k]}
            </option>
          ))}
        </select>
      </div>
      <div className="field">
        <label htmlFor="tAct">
          RACI activity{" "}
          <span className="muted" style={{ fontWeight: 400 }}>
            (optional)
          </span>
        </label>
        <select className="select" id="tAct" value={act} onChange={(e) => setAct(e.target.value)}>
          <option value="">None</option>
          {activities.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
      </div>
      {shown && (
        <p className="small" role="alert" style={{ color: "var(--bad)" }}>
          {shown}
        </p>
      )}
      <div className="row" style={{ justifyContent: "space-between" }}>
        <div className="row">
          <button className="btn primary" type="submit" disabled={busy}>
            {isNew ? "Add task" : "Save"}
          </button>
          <button className="btn" type="button" onClick={onClose}>
            Cancel
          </button>
        </div>
        {!isNew && onDelete && (
          <button className="btn danger sm" type="button" onClick={onDelete} disabled={busy}>
            Delete
          </button>
        )}
      </div>
    </Modal>
  );
}
