import { useState } from "react";
import { Link } from "react-router";
import { errorMessage } from "../api/client";
import { useGenerateRaci, usePlan, useRaci, useSaveRaci } from "../api/hooks";
import type { ProjectBundle, Raci, RaciActivity, RaciValue, Seat } from "../api/types";
import { Icon } from "../components/Icon";
import { useAiOverlay } from "../components/LoadingOverlay";
import { ConfirmModal } from "../components/Modal";
import { useToast } from "../components/Toast";
import { Button, Chip, Notice, PageHead, PageSkeleton, Swatch } from "../components/ui";
import { ProjectPage } from "../layout/ProjectPage";
import { raciSteps } from "../lib/aiSteps";
import { fmtDate } from "../lib/format";
import { nextRaci, raciCheck } from "../lib/raci";

function Key() {
  return (
    <div className="rkey">
      <span>
        <span className="cell R" aria-hidden="true" style={{ display: "inline-grid", placeItems: "center" }}>R</span>Responsible: does the work
      </span>
      <span>
        <span className="cell A" aria-hidden="true" style={{ display: "inline-grid", placeItems: "center" }}>A</span>Accountable: one owner
      </span>
      <span>
        <span className="cell C" aria-hidden="true" style={{ display: "inline-grid", placeItems: "center" }}>C</span>Consulted
      </span>
      <span>
        <span className="cell I" aria-hidden="true" style={{ display: "inline-grid", placeItems: "center" }}>I</span>Informed
      </span>
    </div>
  );
}

function ActivityName({ a, i, owner, onRename, onDelete, startEditing }: {
  a: RaciActivity;
  i: number;
  owner: boolean;
  onRename: (name: string) => void;
  onDelete: () => void;
  startEditing: boolean;
}) {
  const [editing, setEditing] = useState(startEditing);
  const [val, setVal] = useState(a.name);
  const num = <span className="muted mono small">{String(i + 1).padStart(2, "0")}</span>;
  if (editing) {
    const commit = () => {
      setEditing(false);
      const v = val.trim();
      if (v && v !== a.name) onRename(v);
      else setVal(a.name);
    };
    return (
      <span className="row" style={{ gap: 8, flexWrap: "nowrap" }}>
        {num}
        <input
          className="input"
          style={{ padding: "4px 8px" }}
          value={val}
          maxLength={120}
          autoFocus
          aria-label={`Name of activity ${i + 1}`}
          onChange={(e) => setVal(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") commit();
            if (e.key === "Escape") {
              setVal(a.name);
              setEditing(false);
            }
          }}
        />
      </span>
    );
  }
  return (
    <span className="row" style={{ gap: 4, flexWrap: "nowrap", justifyContent: "space-between" }}>
      <span>
        {num}&nbsp; {a.name}
      </span>
      {owner && (
        <span className="row" style={{ gap: 0, flexWrap: "nowrap" }}>
          <button className="btn ghost sm" type="button" aria-label={`Rename ${a.name}`} title="Rename" onClick={() => setEditing(true)}>
            <Icon name="pen" />
          </button>
          <button className="btn ghost sm" type="button" aria-label={`Delete ${a.name}`} title="Delete" onClick={onDelete}>
            ×
          </button>
        </span>
      )}
    </span>
  );
}

function Matrix({ b, seats, raci }: { b: ProjectBundle; seats: Seat[]; raci: Raci }) {
  const p = b.project;
  const owner = b.is_owner;
  const save = useSaveRaci(p.id);
  const gen = useGenerateRaci(p.id);
  const { run, overlay } = useAiOverlay();
  const toast = useToast();
  const [error, setError] = useState("");
  const [confirmRegen, setConfirmRegen] = useState(false);
  const [toDelete, setToDelete] = useState<RaciActivity | null>(null);
  const [fresh, setFresh] = useState<string | null>(null);
  const ids = seats.map((s) => s.id);
  const acts = raci.activities;
  const bad = acts.filter((a) => raciCheck(a, ids)).length;

  const persist = (next: (RaciActivity | (Omit<RaciActivity, "id"> & { id?: string }))[], onDone?: (r: Raci) => void) =>
    save.mutate(next, { onSuccess: (d) => onDone?.(d.raci), onError: (e) => setError(errorMessage(e)) });

  const setCell = (ai: number, seatId: string) => {
    const a = acts[ai]!;
    const v = nextRaci((a.cells[seatId] ?? "") as RaciValue);
    persist(acts.map((x, k) => (k === ai ? { ...x, cells: { ...x.cells, [seatId]: v } } : x)));
  };

  const addActivity = () =>
    persist([...acts, { name: "New activity", cells: {} }], (r) => {
      setFresh(r.activities[r.activities.length - 1]?.id ?? null);
      toast("Activity added. Set one A and at least one R.");
    });

  const regenerate = async () => {
    setConfirmRegen(false);
    setError("");
    try {
      await run("Drafting RACI", raciSteps, () => gen.mutateAsync());
      toast(acts.length ? "RACI draft regenerated" : "RACI draft ready. Click a cell to change it.");
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const holder = (seatId: string) => b.members.find((m) => m.seat_id === seatId);

  if (acts.length === 0) {
    return (
      <>
        <PageHead eyebrow="RACI" title="Responsibility matrix" sub="Who is responsible, accountable, consulted and informed for each activity." />
        <div className="stack" style={{ gap: 16 }}>
          {error && <Notice tone="bad">{error}</Notice>}
          <div className="panel pad empty">
            <Chip tone="ai" icon="spark">
              RACI
            </Chip>
            <h2>No RACI yet.</h2>
            <p className="muted">
              {owner
                ? `The AI drafts 6–10 activities in timeline order up to ${fmtDate(p.deadline)}, with one value per role. You can edit every cell.`
                : "The project owner has not drafted the RACI yet."}
            </p>
            {owner && (
              <div className="row">
                <Button variant="primary" icon="spark" onClick={() => void regenerate()}>
                  Draft RACI with AI
                </Button>
                <Button icon="plus" onClick={addActivity}>
                  Start empty
                </Button>
              </div>
            )}
          </div>
        </div>
        {overlay}
      </>
    );
  }

  return (
    <>
      <PageHead
        eyebrow="RACI"
        title="Responsibility matrix"
        sub={owner ? "Drafted by AI from the project and the role plan. Click a cell to change it." : "Drafted by AI from the project and the role plan. Only the owner can change it."}
        actions={
          owner && (
            <>
              <Button icon="plus" onClick={addActivity} disabled={save.isPending}>
                Add activity
              </Button>
              <Button icon="refresh" onClick={() => setConfirmRegen(true)}>
                Regenerate
              </Button>
            </>
          )
        }
      />
      <div className="stack" style={{ gap: 16 }}>
        {error && <Notice tone="bad">{error}</Notice>}
        <div className="row" style={{ justifyContent: "space-between" }}>
          <Key />
          {bad ? (
            <Chip tone="bad">
              {bad} row{bad > 1 ? "s" : ""} need fixing
            </Chip>
          ) : (
            <Chip tone="ok" icon="check">
              Every activity has one A and at least one R
            </Chip>
          )}
        </div>
        <div className="tablewrap">
          <table className="raci">
            <thead>
              <tr>
                <th className="act">Activity</th>
                {seats.map((s) => (
                  <th key={s.id}>
                    <div className="colhead">
                      <span className="row" style={{ gap: 6, justifyContent: "center" }}>
                        <Swatch color={s.color} />
                        {s.name}
                      </span>
                      <small>{holder(s.id)?.first_name ?? "Unassigned"}</small>
                    </div>
                  </th>
                ))}
                <th>Check</th>
              </tr>
            </thead>
            <tbody>
              {acts.map((a, i) => {
                const c = raciCheck(a, ids);
                return (
                  <tr key={a.id}>
                    <td className="act">
                      <ActivityName
                        a={a}
                        i={i}
                        owner={owner}
                        startEditing={fresh === a.id}
                        onRename={(name) => persist(acts.map((x) => (x.id === a.id ? { ...x, name } : x)))}
                        onDelete={() => setToDelete(a)}
                      />
                    </td>
                    {seats.map((s) => {
                      const v = (a.cells[s.id] ?? "") as RaciValue;
                      const label = `${a.name}, ${s.name}: ${v || "empty"}`;
                      return (
                        <td key={s.id}>
                          {owner ? (
                            <button className={`cell ${v}`} type="button" aria-label={label} onClick={() => setCell(i, s.id)}>
                              {v || "·"}
                            </button>
                          ) : (
                            <span className={`cell ${v}`} role="img" aria-label={label} style={{ display: "inline-grid", placeItems: "center", cursor: "default" }}>
                              {v || "·"}
                            </span>
                          )}
                        </td>
                      );
                    })}
                    <td>
                      {c ? (
                        <Chip tone="bad">{c}</Chip>
                      ) : (
                        <Chip tone="ok" aria-label="OK" icon="check" />
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <p className="small muted">Rows follow the project timeline to {fmtDate(p.deadline)}. Tasks on the board can be linked to an activity.</p>
      </div>
      {confirmRegen && (
        <ConfirmModal
          title="Regenerate the RACI?"
          message="This replaces every activity and value in the matrix. Tasks linked to old activities lose the link."
          confirmLabel="Regenerate"
          danger
          onConfirm={() => void regenerate()}
          onCancel={() => setConfirmRegen(false)}
        />
      )}
      {toDelete && (
        <ConfirmModal
          title="Delete this activity?"
          message={`“${toDelete.name}” will be removed from the matrix. Linked tasks keep their hours.`}
          confirmLabel="Delete"
          danger
          onConfirm={() => {
            persist(acts.filter((x) => x.id !== toDelete.id), () => toast("Activity deleted"));
            setToDelete(null);
          }}
          onCancel={() => setToDelete(null)}
        />
      )}
      {overlay}
    </>
  );
}

function RaciScreen({ b }: { b: ProjectBundle }) {
  const plan = usePlan(b.project.id);
  const raci = useRaci(b.project.id);
  if (plan.error || raci.error) return <Notice tone="bad">{errorMessage(plan.error ?? raci.error)}</Notice>;
  if (!plan.data || !raci.data) return <PageSkeleton blocks={1} />;
  if (plan.data.status !== "approved") {
    return (
      <>
        <PageHead eyebrow="RACI" title="Responsibility matrix" sub="Who is responsible, accountable, consulted and informed for each activity." />
        <div className="panel pad empty">
          <h2>Approve the role plan first.</h2>
          <p className="muted">The RACI uses the approved roles as its columns, so they don't change under your team.</p>
          <Link className="btn primary" to={`/p/${b.project.id}/plan`}>
            Open role plan
          </Link>
        </div>
      </>
    );
  }
  return <Matrix b={b} seats={plan.data.seats} raci={raci.data} />;
}

export function RaciPage() {
  return <ProjectPage active="raci">{(b) => <RaciScreen b={b} />}</ProjectPage>;
}
