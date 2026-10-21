import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { useState } from "react";
import { useSearchParams } from "react-router";
import { ApiError, errorMessage } from "../api/client";
import { useCreateTask, useDeleteTask, useRaci, useTasks, useUpdateTask } from "../api/hooks";
import type { Member, ProjectBundle, Task, TaskInput, TaskStatus } from "../api/types";
import { Icon } from "../components/Icon";
import { STATUS_LABEL, TaskModal } from "../components/TaskModal";
import { useToast } from "../components/Toast";
import { Avatar, Button, Notice, PageHead, PageSkeleton } from "../components/ui";
import { ProjectPage } from "../layout/ProjectPage";
import { fmtH } from "../lib/format";

const COLUMNS: { st: TaskStatus; dot: string }[] = [
  { st: "todo", dot: "var(--muted)" },
  { st: "doing", dot: "var(--accent)" },
  { st: "done", dot: "var(--ok)" },
];

function CardBody({ t, m }: { t: Task; m: Member | undefined }) {
  return (
    <>
      <span className="t">{t.title}</span>
      <span className="meta">
        <span className="by">
          <Avatar person={m} />
          <span>
            {m ? (
              <>
                {m.first_name}
                {m.role_confirmed && m.seat_name && (
                  <>
                    {" · "}
                    <span className="muted">{m.seat_name}</span>
                  </>
                )}
              </>
            ) : (
              <span className="muted">Unassigned</span>
            )}
          </span>
        </span>
        <span className="hrs">{fmtH(t.estimate)} h</span>
      </span>
    </>
  );
}

function Card({ t, m, onOpen }: { t: Task; m: Member | undefined; onOpen: () => void }) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id: t.id, data: { status: t.status } });
  return (
    <div
      ref={setNodeRef}
      className={`card${t.status === "done" ? " done" : ""}${isDragging ? " dragging" : ""}`}
      {...attributes}
      {...listeners}
      role="button"
      aria-roledescription="Draggable task"
      aria-label={`${t.title}, ${m ? m.name : "Unassigned"}, ${fmtH(t.estimate)} hours, ${STATUS_LABEL[t.status]}. Press Enter to edit, Space to move.`}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          onOpen();
          return;
        }
        listeners?.onKeyDown?.(e);
      }}
    >
      <CardBody t={t} m={m} />
    </div>
  );
}

function Column({ st, dot, tasks, members, onOpen, onAdd }: {
  st: TaskStatus;
  dot: string;
  tasks: Task[];
  members: Member[];
  onOpen: (t: Task) => void;
  onAdd: () => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: st });
  const hrs = tasks.reduce((a, t) => a + t.estimate, 0);
  return (
    <section ref={setNodeRef} className={`col${isOver ? " over" : ""}`} aria-label={STATUS_LABEL[st]}>
      <div className="colhead">
        <h3>
          <span className="dot" style={{ color: dot }} />
          {STATUS_LABEL[st]} <span className="muted mono small">{tasks.length}</span>
        </h3>
        <span className="hrs">{fmtH(hrs)} h</span>
      </div>
      {tasks.map((t) => (
        <Card key={t.id} t={t} m={members.find((m) => m.id === t.assignee_id)} onOpen={() => onOpen(t)} />
      ))}
      {st === "todo" && (
        <button className="btn ghost sm" type="button" style={{ justifyContent: "flex-start" }} onClick={onAdd}>
          <Icon name="plus" />
          Add task
        </button>
      )}
    </section>
  );
}

const keyboardCodes = { start: ["Space"], cancel: ["Escape"], end: ["Space", "Enter"] };

function Board({ b }: { b: ProjectBundle }) {
  const p = b.project;
  const tasksQ = useTasks(p.id);
  const raci = useRaci(p.id);
  const create = useCreateTask(p.id);
  const update = useUpdateTask(p.id);
  const del = useDeleteTask(p.id);
  const toast = useToast();
  const [params, setParams] = useSearchParams();
  const filter = params.get("who") ?? "all";
  const [modal, setModal] = useState<{ task: Task | null } | null>(null);
  const [modalError, setModalError] = useState("");
  const [dragId, setDragId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 180, tolerance: 6 } }),
    useSensor(KeyboardSensor, { keyboardCodes }),
  );

  if (tasksQ.error) return <Notice tone="bad">{errorMessage(tasksQ.error)}</Notice>;
  if (!tasksQ.data) return <PageSkeleton blocks={1} />;
  const all = tasksQ.data;
  const shown = all.filter((t) => filter === "all" || (filter === "none" ? !t.assignee_id : t.assignee_id === filter));
  const setFilter = (f: string) => setParams(f === "all" ? {} : { who: f }, { replace: true });

  const move = (id: string, status: TaskStatus) => {
    const t = all.find((x) => x.id === id);
    if (!t || t.status === status) return;
    update.mutate({ taskId: id, patch: { status } }, { onSuccess: () => toast(`Moved to ${STATUS_LABEL[status]}`), onError: (e) => setError(errorMessage(e)) });
  };
  const onDragStart = (e: DragStartEvent) => setDragId(String(e.active.id));
  const onDragEnd = (e: DragEndEvent) => {
    setDragId(null);
    if (e.over) move(String(e.active.id), e.over.id as TaskStatus);
  };

  const close = () => {
    setModal(null);
    setModalError("");
  };
  const onSave = (input: TaskInput) => {
    const fail = (e: unknown) => setModalError(e instanceof ApiError ? (Object.values(e.fields)[0] ?? e.message) : errorMessage(e));
    if (!modal?.task) {
      create.mutate(input, { onSuccess: () => (close(), toast("Task added")), onError: fail });
    } else {
      update.mutate({ taskId: modal.task.id, patch: input }, { onSuccess: () => (close(), toast("Task saved")), onError: fail });
    }
  };
  const onDelete = () => {
    if (!modal?.task) return;
    del.mutate(modal.task.id, { onSuccess: () => toast("Task deleted"), onError: (e) => setError(errorMessage(e)) });
    close();
  };

  const dragged = dragId ? all.find((t) => t.id === dragId) : undefined;
  const memberOf = (t: Task) => b.members.find((m) => m.id === t.assignee_id);

  return (
    <>
      <PageHead
        eyebrow="Board"
        title="Task board"
        sub="Drag cards between columns, or click a card to edit it."
        actions={
          <Button variant="primary" icon="plus" onClick={() => setModal({ task: null })}>
            New task
          </Button>
        }
      />
      <div className="stack" style={{ gap: 16 }}>
        {error && <Notice tone="bad">{error}</Notice>}
        {all.length === 0 && (
          <Notice tone="ai">
            <b>No tasks yet.</b> Add tasks with an hour estimate; the dashboard uses the estimates to show who is overloaded.
          </Notice>
        )}
        <div className="filters" role="group" aria-label="Filter by person">
          <button type="button" className="all" aria-pressed={filter === "all"} onClick={() => setFilter("all")}>
            Everyone
          </button>
          {b.members.map((m) => (
            <button type="button" key={m.id} aria-pressed={filter === m.id} onClick={() => setFilter(m.id)}>
              <Avatar person={m} />
              {m.first_name}
            </button>
          ))}
          <button type="button" className="all" aria-pressed={filter === "none"} onClick={() => setFilter("none")}>
            Unassigned
          </button>
        </div>
        <DndContext sensors={sensors} onDragStart={onDragStart} onDragEnd={onDragEnd} onDragCancel={() => setDragId(null)}>
          <div className="board">
            {COLUMNS.map((c) => (
              <Column
                key={c.st}
                st={c.st}
                dot={c.dot}
                tasks={shown.filter((t) => t.status === c.st)}
                members={b.members}
                onOpen={(t) => setModal({ task: t })}
                onAdd={() => setModal({ task: null })}
              />
            ))}
          </div>
          <DragOverlay dropAnimation={null}>
            {dragged ? (
              <div className={`card${dragged.status === "done" ? " done" : ""}`} style={{ cursor: "grabbing", boxShadow: "var(--shadow)" }}>
                <CardBody t={dragged} m={memberOf(dragged)} />
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      </div>
      {modal && (
        <TaskModal
          task={modal.task}
          members={b.members}
          activities={raci.data?.activities ?? []}
          defaultAssignee={filter !== "all" && filter !== "none" ? filter : null}
          serverError={modalError}
          busy={create.isPending || update.isPending}
          onSave={onSave}
          onDelete={onDelete}
          onClose={close}
        />
      )}
    </>
  );
}

export function BoardPage() {
  return <ProjectPage active="board">{(b) => <Board b={b} />}</ProjectPage>;
}
