import { useState } from "react";
import { Link } from "react-router";
import { errorMessage } from "../api/client";
import { useAcceptMatch, useAssignSeat, useMatching, useMe, usePlan, useRunMatching } from "../api/hooks";
import type { MatchRow, Matching, Member, ProjectBundle, Seat } from "../api/types";
import { useAiOverlay } from "../components/LoadingOverlay";
import { ConfirmModal } from "../components/Modal";
import { useToast } from "../components/Toast";
import { Avatar, Button, Chip, FitBar, Notice, PageHead, PageSkeleton } from "../components/ui";
import { ProjectPage } from "../layout/ProjectPage";
import { matchingSteps } from "../lib/aiSteps";
import { fmtDate, shortName, todayISO } from "../lib/format";
import { MSG } from "../lib/validate";

function ReadyList({ members, teamSize }: { members: Member[]; teamSize: number }) {
  const ready = members.filter((m) => m.has_profile).length;
  return (
    <div className="panel pad stack" style={{ gap: 6 }}>
      <div className="row" style={{ justifyContent: "space-between" }}>
        <h3>Profiles</h3>
        <Chip tone={ready === members.length && members.length === teamSize ? "ok" : "warn"} className="mono">
          {ready} / {teamSize}
        </Chip>
      </div>
      <div className="ready">
        {members.map((m) => (
          <div key={m.id}>
            <Avatar person={m} />
            <span style={{ flex: 1 }}>
              {m.name}
              {m.is_owner && <span className="small muted"> · owner</span>}
            </span>
            {m.has_profile ? (
              <Chip tone="ok" icon="check">
                Ready
              </Chip>
            ) : (
              <Chip tone="warn">No profile</Chip>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function ranLabel(ranAt: string | null): string {
  if (!ranAt) return "Roles set by the team";
  const d = todayISO(new Date(ranAt));
  return d === todayISO() ? "AI matching ran today" : `AI matching ran on ${fmtDate(d)}`;
}

function Badge({ row }: { row: MatchRow }) {
  if (!row.user_id) return <Chip tone="line">Unassigned</Chip>;
  if (row.state === "accepted")
    return row.source === "manual" ? (
      <Chip icon="check">Set manually</Chip>
    ) : (
      <Chip tone="ok" icon="check">
        Accepted
      </Chip>
    );
  return (
    <Chip tone="ai" icon="spark">
      Suggested
    </Chip>
  );
}

type RowProps = {
  row: MatchRow;
  seat: Seat;
  seats: Seat[];
  members: Member[];
  owner: boolean;
  onAssign: (seatId: string, userId: string) => void;
  onAccept: (seatId: string) => void;
  busy: boolean;
};

function MatchRowView({ row, seat, seats, members, owner, onAssign, onAccept, busy }: RowProps) {
  const m = members.find((x) => x.id === row.user_id) ?? null;
  const manual = row.source === "manual";
  const runner = seats.find((s) => s.id === row.runner_up_seat_id);
  const fit = manual || !m || row.fit === null ? null : row.fit;
  return (
    <div className="match">
      <div className="slotname">
        <span className="swatch" style={{ background: `var(${seat.color})`, marginTop: 6 }} />
        <div>
          <b>{seat.name}</b>
          <span className="small muted">{seat.effort_share}% of effort</span>
        </div>
      </div>
      <div className="stack" style={{ gap: 10 }}>
        <div className="who">
          <Avatar person={m} />
          {owner ? (
            <select aria-label={`Person for ${seat.name}`} value={row.user_id ?? ""} disabled={busy} onChange={(e) => e.target.value && onAssign(seat.id, e.target.value)}>
              {!row.user_id && (
                <option value="" disabled>
                  Unassigned
                </option>
              )}
              {members.map((x) => (
                <option key={x.id} value={x.id}>
                  {shortName(x.name)}
                </option>
              ))}
            </select>
          ) : (
            <span>{m ? shortName(m.name) : <span className="muted">Unassigned</span>}</span>
          )}
        </div>
        <div className="fit">
          <div className="row small" style={{ justifyContent: "space-between" }}>
            <span className="muted">Fit</span>
            <b className="mono">{fit ?? "—"}</b>
          </div>
          <FitBar value={fit ?? 0} />
        </div>
      </div>
      <div className="expl">
        {!m ? (
          <p>
            <span className="muted">No one holds this role yet.{owner ? " Choose a person to assign it." : ""}</span>
          </p>
        ) : manual ? (
          <p>
            <span className="muted">Changed by the team. The AI explanation no longer applies.</span>
          </p>
        ) : (
          <>
            <p>{row.explanation}</p>
            {runner && <span className="small muted">Also a good fit: {runner.name}</span>}
          </>
        )}
      </div>
      <div className="src">
        <Badge row={row} />
        {row.outdated && m && <Chip tone="warn">Outdated</Chip>}
        {owner && m && row.state !== "accepted" && (
          <Button size="sm" onClick={() => onAccept(seat.id)} disabled={busy}>
            Accept
          </Button>
        )}
      </div>
    </div>
  );
}

function NotRun({ b, matching, onRun, error }: { b: ProjectBundle; matching: Matching; onRun: () => void; error: string }) {
  const r = matching.readiness;
  const me = useMe().data;
  const p = b.project;
  const iMissing = b.members.some((m) => m.id === me?.id && !m.has_profile);
  let title = "Everyone is ready.";
  let text = `The AI assigns each of the ${p.seat_count} roles to one person and explains why. Nothing is final until you accept it.`;
  if (!r.plan_approved) {
    title = "Approve the role plan first.";
    text = "Matching needs an approved plan so the roles don't change under your team.";
  } else if (r.member_count < 3) {
    title = "Invite your team first.";
    text = `Matching needs at least 3 members. ${r.member_count} of ${p.team_size} have joined; share the invite code from the role plan.`;
  } else if (r.missing_profiles.length > 0) {
    title = "Waiting for profiles.";
    text = `Still needs a profile: ${r.missing_profiles.join(", ")}. The AI reads every profile before it suggests roles.`;
  } else if (!b.is_owner) {
    text = "The project owner runs the matching. You will see the suggestions here.";
  }
  return (
    <>
      <PageHead eyebrow="Team roles" title="Match people to roles" sub="The AI reads every profile and finds the best role for each person across the whole team." />
      <div className="stack" style={{ gap: 16 }}>
        {error && <Notice tone="bad">{error}</Notice>}
        <div className="split">
          <div className="panel pad stack" style={{ alignItems: "flex-start" }}>
            <Chip tone="ai" icon="spark">
              AI matching
            </Chip>
            <h2>{title}</h2>
            <p className="muted">{text}</p>
            {!r.plan_approved ? (
              <Link className="btn primary" to={`/p/${p.id}/plan`}>
                Open role plan
              </Link>
            ) : r.can_run && b.is_owner ? (
              <Button variant="primary" icon="spark" onClick={onRun}>
                Run AI matching
              </Button>
            ) : iMissing ? (
              <Link className="btn primary" to="/profile">
                Fill in my profile
              </Link>
            ) : null}
          </div>
          <ReadyList members={b.members} teamSize={p.team_size} />
        </div>
      </div>
    </>
  );
}

function RolesScreen({ b }: { b: ProjectBundle }) {
  const p = b.project;
  const q = useMatching(p.id);
  const plan = usePlan(p.id);
  const runM = useRunMatching(p.id);
  const accept = useAcceptMatch(p.id);
  const assign = useAssignSeat(p.id);
  const { run, overlay } = useAiOverlay();
  const toast = useToast();
  const [error, setError] = useState("");
  const [manual, setManual] = useState(false);
  const [confirmRerun, setConfirmRerun] = useState(false);

  if (q.error || plan.error) return <Notice tone="bad">{errorMessage(q.error ?? plan.error)}</Notice>;
  if (!q.data || !plan.data) return <PageSkeleton blocks={2} />;
  const M = q.data;
  const seats = plan.data.seats;

  const doRun = async () => {
    setConfirmRerun(false);
    setError("");
    try {
      await run("Matching people to roles", matchingSteps(b.members.length), () => runM.mutateAsync(undefined));
      setManual(false);
      toast("Matching done. Review and accept.");
    } catch (e) {
      const msg = errorMessage(e);
      setError(msg);
      // Profiles are kept; let the owner assign every seat by hand (spec 6.6).
      if (msg === MSG.aiUnavailable) setManual(true);
    }
  };

  const onAssign = (seatId: string, userId: string) => {
    const swapped = M.rows.some((r) => r.user_id === userId && r.seat_id !== seatId) && M.rows.some((r) => r.seat_id === seatId && r.user_id);
    assign.mutate(
      { seatId, userId },
      { onSuccess: () => toast(swapped ? "Roles swapped and saved as manual" : "Role saved as manual"), onError: (e) => setError(errorMessage(e)) },
    );
  };
  const onAccept = (seatId: string) => accept.mutate(seatId, { onError: (e) => setError(errorMessage(e)) });

  if (!M.ran && !manual)
    return (
      <>
        <NotRun b={b} matching={M} onRun={() => void doRun()} error={error} />
        {overlay}
      </>
    );

  const accepted = M.rows.filter((r) => r.user_id && r.state === "accepted").length;
  const acceptable = M.rows.some((r) => r.user_id && r.state !== "accepted");
  const outdated = M.rows.filter((r) => r.outdated && r.user_id).map((r) => b.members.find((m) => m.id === r.user_id)?.first_name).filter(Boolean);
  const busy = assign.isPending || accept.isPending;
  const sub = `${manual && !M.ran ? "Assign roles by hand" : ranLabel(M.ran_at)} · ${accepted} of ${M.rows.length} roles confirmed.`;

  return (
    <>
      <PageHead
        eyebrow="Team roles"
        title="Who does what"
        sub={sub}
        actions={
          b.is_owner && (
            <>
              <Button icon="refresh" disabled={!M.readiness.can_run} onClick={() => (M.rows.some((r) => r.state === "accepted" && r.user_id) ? setConfirmRerun(true) : void doRun())}>
                Run again
              </Button>
              <Button variant="primary" icon="check" disabled={!acceptable || busy} onClick={() => accept.mutate(null, { onSuccess: () => toast("All roles confirmed"), onError: (e) => setError(errorMessage(e)) })}>
                Accept all
              </Button>
            </>
          )
        }
      />
      <div className="stack" style={{ gap: 16 }}>
        {error && <Notice tone="bad">{error}</Notice>}
        {outdated.length > 0 && (
          <Notice tone="warn">
            <b>Profiles changed:</b> {outdated.join(", ")} updated their profile after matching ran. {b.is_owner ? "Run again to use the new text." : "The owner can run matching again."}
          </Notice>
        )}
        {M.gaps.length > 0 && (
          <Notice tone="warn">
            <b>Skill gap:</b> no one in the team listed {M.gaps.map((g) => g.skill).join(", ")}. {M.gaps.map((g) => g.reason).join(" ")}
          </Notice>
        )}
        {M.growth_note && (
          <Notice tone="ai">
            <b>Growth note.</b> {M.growth_note}
          </Notice>
        )}
        <div className="panel" style={{ overflow: "hidden" }}>
          {M.rows.map((r) => {
            const seat = seats.find((s) => s.id === r.seat_id);
            return seat ? (
              <MatchRowView key={r.seat_id} row={r} seat={seat} seats={seats} members={b.members} owner={b.is_owner} onAssign={onAssign} onAccept={onAccept} busy={busy} />
            ) : null;
          })}
        </div>
        <p className="small muted">Changing a person swaps them with whoever held that role. Every change is saved as “set manually”.</p>
      </div>
      {confirmRerun && (
        <ConfirmModal
          title="Run matching again?"
          message="This replaces every suggestion, including the roles your team already accepted."
          confirmLabel="Run again"
          onConfirm={() => void doRun()}
          onCancel={() => setConfirmRerun(false)}
        />
      )}
      {overlay}
    </>
  );
}

export function RolesPage() {
  return <ProjectPage active="roles">{(b) => <RolesScreen b={b} />}</ProjectPage>;
}

