import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { ApiError, errorMessage } from "../api/client";
import { useLogin, useRegister } from "../api/hooks";
import { Field, Input } from "../components/Field";
import { Icon } from "../components/Icon";
import { useToast } from "../components/Toast";
import { safeNext } from "../lib/nav";
import { validateRegistration, type Errors } from "../lib/validate";

type Tab = "login" | "register";

export function LoginPage() {
  const [tab, setTab] = useState<Tab>("login");
  const [v, setV] = useState({ full_name: "", email: "", password: "", consent: false });
  const [errors, setErrors] = useState<Errors>({});
  const [formError, setFormError] = useState("");
  const login = useLogin();
  const register = useRegister();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const toast = useToast();
  const reg = tab === "register";
  const busy = login.isPending || register.isPending;

  const set = (k: keyof typeof v) => (val: string | boolean) => {
    setV((s) => ({ ...s, [k]: val }));
    if (errors[k]) setErrors((e) => ({ ...e, [k]: "" }));
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    setFormError("");
    const done = (msg: string) => () => {
      navigate(safeNext(params.get("next")), { replace: true });
      toast(msg);
    };
    const fail = (err: unknown) => {
      if (err instanceof ApiError && Object.keys(err.fields).length) setErrors(err.fields);
      else setFormError(errorMessage(err));
    };
    if (reg) {
      const errs = validateRegistration(v);
      setErrors(errs);
      if (Object.values(errs).some(Boolean)) return;
      register.mutate(v, { onSuccess: done("Account created"), onError: fail });
    } else {
      if (!v.email.trim() || !v.password) {
        setFormError("Enter your email and password.");
        return;
      }
      login.mutate({ email: v.email, password: v.password }, { onSuccess: done("Logged in"), onError: fail });
    }
  };

  const switchTab = (t: Tab) => {
    setTab(t);
    setErrors({});
    setFormError("");
  };

  return (
    <div className="auth">
      <section className="auth-side">
        <div className="brand">
          <Icon name="logo" />
          <span>TeamLoad</span>
        </div>
        <div className="stack" style={{ gap: 28 }}>
          <h1>
            Describe the project.
            <br />
            Get the team it needs.
          </h1>
          <div className="flowline">
            <div>
              <span className="n">1</span>
              <p>
                <b>Start a project.</b> Write what you build, set the deadline and the team size.
              </p>
            </div>
            <div>
              <span className="n">2</span>
              <p>
                <b>AI designs the roles.</b> One role per seat, with responsibilities and skills for this project.
              </p>
            </div>
            <div>
              <span className="n">3</span>
              <p>
                <b>Members join and describe themselves.</b> AI matches each person to a role and explains why.
              </p>
            </div>
            <div>
              <span className="n">4</span>
              <p>
                <b>Track the work fairly.</b> RACI, task board and a workload dashboard that shows who is overloaded.
              </p>
            </div>
          </div>
        </div>
        <p className="small" style={{ opacity: 0.6 }}>
          The AI suggests. Your team decides.
        </p>
      </section>
      <section className="auth-form">
        <form onSubmit={onSubmit} noValidate>
          <div className="tabs" role="tablist">
            <button type="button" role="tab" aria-selected={!reg} onClick={() => switchTab("login")}>
              Log in
            </button>
            <button type="button" role="tab" aria-selected={reg} onClick={() => switchTab("register")}>
              Create account
            </button>
          </div>
          <h2>{reg ? "Create your account" : "Welcome back"}</h2>
          {reg && (
            <Field label="Full name" error={errors.full_name}>
              {(p) => <Input {...p} value={v.full_name} onChange={(e) => set("full_name")(e.target.value)} autoComplete="name" required />}
            </Field>
          )}
          <Field label="Email" hint={reg ? "A university email is preferred, but any email works." : undefined} error={errors.email}>
            {(p) => <Input {...p} type="email" value={v.email} onChange={(e) => set("email")(e.target.value)} autoComplete="email" required />}
          </Field>
          <Field label="Password" hint={reg ? "At least 8 characters." : undefined} error={errors.password}>
            {(p) => (
              <Input
                {...p}
                type="password"
                value={v.password}
                onChange={(e) => set("password")(e.target.value)}
                autoComplete={reg ? "new-password" : "current-password"}
                required
                minLength={8}
              />
            )}
          </Field>
          {reg && (
            <>
              <label className="check" htmlFor="fConsent">
                <input
                  type="checkbox"
                  id="fConsent"
                  checked={v.consent}
                  onChange={(e) => set("consent")(e.target.checked)}
                  aria-invalid={errors.consent ? true : undefined}
                  aria-describedby={errors.consent ? "consentErr" : undefined}
                />
                <span>
                  I agree that my profile text and project descriptions are sent to an AI service to suggest roles. I can delete my
                  profile at any time.
                </span>
              </label>
              {errors.consent && (
                <p className="small" id="consentErr" style={{ color: "var(--bad)" }}>
                  {errors.consent}
                </p>
              )}
            </>
          )}
          {formError && (
            <p className="small" role="alert" style={{ color: "var(--bad)" }}>
              {formError}
            </p>
          )}
          <button className="btn primary" type="submit" style={{ justifyContent: "center" }} disabled={busy}>
            {reg ? "Create account" : "Log in"}
          </button>
          <p className="small muted" style={{ textAlign: "center" }}>
            {reg ? "Already have an account? Use the Log in tab." : "New here? Use the Create account tab."}
          </p>
        </form>
      </section>
    </div>
  );
}
