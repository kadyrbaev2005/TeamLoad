import { Link } from "react-router";
import { Icon } from "../components/Icon";

/** 404 / no access, in the join card's visual language. */
export function NotFoundPage({ message }: { message?: string }) {
  return (
    <div className="joinwrap">
      <div className="panel joincard">
        <div className="brand" style={{ padding: 0 }}>
          <Icon name="logo" />
          <span>TeamLoad</span>
        </div>
        <div className="stack" style={{ gap: 6 }}>
          <span className="eyebrow">404</span>
          <h1>Page not found</h1>
          <p className="muted">{message ?? "This page does not exist, or you are not a member of this project."}</p>
        </div>
        <div className="row">
          <Link className="btn primary" to="/projects">
            Go to my projects
          </Link>
        </div>
      </div>
    </div>
  );
}
