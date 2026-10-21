import { useState, type FormEvent, type ReactNode } from "react";
import { Link, Navigate, useNavigate, useParams } from "react-router";
import { errorMessage, isStatus } from "../api/client";
import { useInvite, useJoin } from "../api/hooks";
import { Icon } from "../components/Icon";
import { useToast } from "../components/Toast";
import { Notice, Skeleton } from "../components/ui";
import { fmtDate } from "../lib/format";

function JoinCard({ children }: { children: ReactNode }) {
  return (
    <div className="joinwrap">
      <div className="panel joincard">
        <div className="brand" style={{ padding: 0 }}>
          <Icon name="logo" />
          <span>TeamLoad</span>
        </div>
        {children}
      </div>
    </div>
  );
}

/** Code input card for /join (and after an invalid code). */
function CodeForm({ error, initial = "" }: { error?: string; initial?: string }) {
  const [code, setCode] = useState(initial);
  const navigate = useNavigate();
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const c = code.trim().toUpperCase();
    if (c) navigate(`/join/${encodeURIComponent(c)}`);
  };
  return (
    <form className="stack" onSubmit={onSubmit} noValidate>
      <div className="stack" style={{ gap: 6 }}>
        <span className="eyebrow">Join a project</span>
        <h1>Enter your invite code</h1>
        <p className="muted">Your project owner can copy the 6-character code from the role plan or the settings page.</p>
      </div>
      {error && <Notice tone="bad">{error}</Notice>}
      <div className="field">
        <label htmlFor="jCode">Invite code</label>
        <input
          className="input mono"
          id="jCode"
          value={code}
          maxLength={6}
          autoCapitalize="characters"
          autoComplete="off"
          spellCheck={false}
          style={{ textTransform: "uppercase", letterSpacing: ".12em", fontSize: 18 }}
          onChange={(e) => setCode(e.target.value)}
          aria-invalid={error ? true : undefined}
        />
      </div>
      <div className="row">
        <button className="btn primary" type="submit" disabled={!code.trim()}>
          Continue
        </button>
        <Link className="btn ghost" to="/projects">
          Cancel
        </Link>
      </div>
    </form>
  );
}

export function JoinPage() {
  const { code = "" } = useParams();
  const invite = useInvite(code.toUpperCase());
  const join = useJoin();
  const navigate = useNavigate();
  const toast = useToast();
  const [joinError, setJoinError] = useState("");

  if (!code) {
    return (
      <JoinCard>
        <CodeForm />
      </JoinCard>
    );
  }
  if (invite.isPending) {
    return (
      <JoinCard>
        <div className="stack" aria-busy="true">
          <Skeleton kind="line" style={{ width: 160 }} />
          <Skeleton kind="title" />
          <Skeleton kind="block" style={{ height: 120 }} />
        </div>
      </JoinCard>
    );
  }
  if (invite.error) {
    return (
      <JoinCard>
        <CodeForm error={isStatus(invite.error, 404) ? "Invalid invite code" : errorMessage(invite.error)} initial={code.toUpperCase()} />
      </JoinCard>
    );
  }
  const p = invite.data;
  if (p.is_member) return <Navigate to={`/p/${p.project_id}/plan`} replace />;

  const onJoin = () => {
    setJoinError("");
    join.mutate(p.code, {
      onSuccess: (r) => {
        if (r.already_member) {
          navigate(`/p/${r.project_id}/plan`);
          return;
        }
        if (r.profile_ready) {
          navigate(`/p/${r.project_id}/plan`);
          toast(`You joined ${r.title}.`);
        } else {
          navigate("/profile");
          toast(`You joined ${r.title}. Fill in your profile next.`);
        }
      },
      onError: (e) => setJoinError(errorMessage(e)),
    });
  };

  return (
    <JoinCard>
      <div className="stack" style={{ gap: 6 }}>
        <span className="eyebrow">You're invited to join</span>
        <h1>{p.title}</h1>
        <p className="muted">
          {p.context ? `${p.context} · created by ${p.owner_name}` : `Created by ${p.owner_name}`}
        </p>
      </div>
      <p>{p.description_excerpt}</p>
      <div className="grid2">
        <div className="panel pad" style={{ padding: "12px 14px" }}>
          <span className="eyebrow">Deadline</span>
          <p>
            <b>{fmtDate(p.deadline)}</b>
          </p>
        </div>
        <div className="panel pad" style={{ padding: "12px 14px" }}>
          <span className="eyebrow">Team</span>
          <p>
            <b className="mono">
              {p.member_count} / {p.team_size}
            </b>{" "}
            joined
          </p>
        </div>
      </div>
      <div className="stack" style={{ gap: 8 }}>
        {p.open_roles.length > 0 && (
          <>
            <span className="eyebrow">Open roles in this project</span>
            <div className="skills">
              {p.open_roles.map((r) => (
                <span className="chip line" key={r.name}>
                  <span className="swatch" style={{ background: `var(${r.color})` }} />
                  {r.name}
                </span>
              ))}
            </div>
          </>
        )}
        <p className="small muted">After you join and fill in your profile, the AI suggests which role fits you best.</p>
      </div>
      {joinError && <Notice tone="bad">{joinError}</Notice>}
      <div className="row">
        <button className="btn primary" type="button" onClick={onJoin} disabled={join.isPending}>
          Join {p.title}
        </button>
        <span className="small muted mono">Code {p.code}</span>
      </div>
    </JoinCard>
  );
}
