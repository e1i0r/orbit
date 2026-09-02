package db

// Every statement this package sends, in one place.
//
// The alternative — a literal at each call site — reads fine one function at
// a time and badly all at once: whether two writes agree about a column, or
// whether a read is ordered the way another read is, are questions about the
// set of them, and they can only be asked of a diff if the set is somewhere
// you can look at it.
//
// The schema itself is not here. It is the shape rather than a statement
// against it, and it lives beside the migration that applies it in
// schema.go.
//
// Why a statement is written the way it is stays at the call site, where the
// decision is. What is here is what it does.

// The version stamped in the file, which is how a migration knows where to
// start. The write is a format string because PRAGMA takes no parameters.
const (
	readVersion  = `PRAGMA user_version`
	stampVersion = `PRAGMA user_version = %d`
)

// Tasks. A task appears the first time anything mentions it and is described
// when task.created arrives, which can be later than the first mention.
const (
	insertTask = `INSERT INTO task(task_id, created_at) VALUES(?,?) ON CONFLICT(task_id) DO NOTHING`
	fillTask   = `UPDATE task SET text = ?, flow = ?, created_at = ? WHERE task_id = ?`
	findTask   = `SELECT id FROM task WHERE task_id = ?`
)

// Repositories, and the tasks worked in them. Both inserts are written to be
// harmless the second time: repo.joined is appended whenever a worktree is
// opened, which happens again on every retry.
const (
	insertRepo = `INSERT INTO repo(abs_path, name, first_seen) VALUES(?,?,?) ON CONFLICT(abs_path) DO NOTHING`

	joinTaskToRepo = `INSERT INTO task_repo(task_id, repo_id, joined_at)
	                  SELECT ?, id, ? FROM repo WHERE abs_path = ?
	                  ON CONFLICT(task_id, repo_id) DO NOTHING`

	selectTasksOfRepo = `SELECT t.task_id
	                       FROM task t
	                       JOIN task_repo tr ON tr.task_id = t.id
	                       JOIN repo r ON r.id = tr.repo_id
	                      WHERE r.abs_path = ?
	                        AND NOT EXISTS (SELECT 1 FROM event e
	                                         WHERE e.task_id = t.id AND e.kind = ?)
	                      ORDER BY t.task_id`

	selectReposOfTask = `SELECT r.abs_path
	                       FROM repo r
	                       JOIN task_repo tr ON tr.repo_id = r.id
	                       JOIN task t ON t.id = tr.task_id
	                      WHERE t.task_id = ?
	                      ORDER BY tr.joined_at, r.abs_path`

	selectTasksAndRepos = `SELECT t.task_id, r.abs_path, r.name
	                         FROM task t
	                         JOIN task_repo tr ON tr.task_id = t.id
	                         JOIN repo r ON r.id = tr.repo_id
	                        WHERE NOT EXISTS (SELECT 1 FROM event e
	                                           WHERE e.task_id = t.id AND e.kind = ?)
	                        ORDER BY t.task_id, tr.joined_at, r.abs_path`

	unjoinRepo = `DELETE FROM task_repo
	                    WHERE repo_id IN (SELECT id FROM repo WHERE abs_path = ?)`
)

// Runs. An attempt is numbered by how many came before it, and it is ended
// by whichever event ended it.
const (
	countRuns = `SELECT count(*) FROM run WHERE task_id = ?`
	insertRun = `INSERT INTO run(task_id, n, started_at) VALUES(?,?,?)`
	latestRun = `SELECT id FROM run WHERE task_id = ? ORDER BY n DESC LIMIT 1`
	endRun    = `UPDATE run SET ended_at = ?, outcome = ? WHERE id = ? AND ended_at IS NULL`
)

// Phases. The last one is how a run that ended takes its phases with it.
const (
	countPhases   = `SELECT count(*) FROM phase WHERE run_id = ?`
	insertPhase   = `INSERT INTO phase(run_id, n, name, engine, model, started_at) VALUES(?,?,?,?,?,?)`
	runningPhase  = `SELECT id FROM phase WHERE run_id = ? AND ended_at IS NULL ORDER BY n DESC LIMIT 1`
	endPhase      = `UPDATE phase SET ended_at = ?, outcome = ? WHERE id = ?`
	endOpenPhases = `UPDATE phase SET ended_at = ?, outcome = ? WHERE run_id = ? AND ended_at IS NULL`
)

// The supervisor thread. The subselects turn a task id or a repository path
// into the row it names, and into NULL when it names none — a turn about a
// task the record has never heard of is still a turn that was taken.
const (
	insertMessage = `INSERT INTO message(at, kind, source, who, task_id, repo_id, text, data)
	                 VALUES(?,?,?,?,
	                        (SELECT id FROM task WHERE task_id = ?),
	                        (SELECT id FROM repo WHERE abs_path = ?),
	                        ?,?)`

	retractMessage = `UPDATE message SET retracted_at = ? WHERE at = ? AND retracted_at IS NULL`

	selectMessages = `SELECT kind, at, text, data FROM message ORDER BY id`
)

// How much of each thing the record already holds. Both are appended to and
// never reordered, so a count is a position — which is what a migration
// reads to know where it got to.
const (
	countEvents   = `SELECT count(*) FROM event e JOIN task t ON t.id = e.task_id WHERE t.task_id = ?`
	countMessages = `SELECT count(*) FROM message`
)

// Events.
const insertEvent = `INSERT INTO event(task_id, run_id, phase_id, kind, at, phase, text, data)
                     VALUES(?,?,?,?,?,?,?,?)`

// Reading back. Both of these are ordered by the row's own id or number and
// never by a timestamp: the clocks of two processes have no order between
// them, and one machine's clock stepping backwards would otherwise reorder a
// history that did not change.
const (
	selectEvents = `SELECT e.id, e.kind, e.at, e.phase, e.text, e.data
	                  FROM event e JOIN task t ON t.id = e.task_id
	                 WHERE t.task_id = ?
	                 ORDER BY e.id`

	selectTasks = `SELECT task_id FROM task ORDER BY id`

	selectSince = `SELECT e.id, t.task_id, e.kind, e.at, e.phase, e.text, e.data
	                 FROM event e JOIN task t ON t.id = e.task_id
	                WHERE e.id > ?
	                ORDER BY e.id`

	selectSinceOf = `SELECT e.id, t.task_id, e.kind, e.at, e.phase, e.text, e.data
	                   FROM event e JOIN task t ON t.id = e.task_id
	                  WHERE e.id > ? AND t.task_id = ?
	                  ORDER BY e.id`

	selectRuns = `SELECT r.n, r.started_at, r.ended_at, r.outcome,
	                     p.n, p.name, p.engine, p.model, p.started_at, p.ended_at, p.outcome
	                FROM run r
	                JOIN task t ON t.id = r.task_id
	                LEFT JOIN phase p ON p.run_id = r.id
	               WHERE t.task_id = ?
	               ORDER BY r.n, p.n`
)

// What SQLite says about its own file, and how much of ours is in it.
//
// The check is a pragma rather than a query because that is the only thing
// that reads the pages themselves; everything else here would read an index
// and believe it. The counts are one statement so that the three numbers
// answer for one moment rather than for three.
const (
	integrityCheck = `PRAGMA integrity_check`

	countAll = `SELECT (SELECT count(*) FROM task),
	                   (SELECT count(*) FROM event),
	                   (SELECT count(*) FROM message)`
)
