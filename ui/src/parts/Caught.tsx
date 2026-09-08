// What is shown when a screen throws.
//
// React unmounts the whole tree when a render throws, so without this one
// bad line of one file leaves a blank page and no way back — which is what
// happened, and the reader could not tell a crash from a server that had
// stopped. This keeps the rest of the window standing, says what broke, and
// offers the two ways out: draw it again, or go back to the board.

import { Component, type ErrorInfo, type ReactNode } from "react";

interface State {
  said?: string;
}

export class Caught extends Component<{ children: ReactNode; back?: () => void }, State> {
  state: State = {};

  static getDerivedStateFromError(err: unknown): State {
    return { said: err instanceof Error ? err.message : String(err) };
  }

  componentDidCatch(err: Error, where: ErrorInfo) {
    // The console is where a reader can copy it from, and where it is worth
    // having when this is reported.
    console.error("orbit: a screen threw", err, where.componentStack);
  }

  render() {
    if (this.state.said === undefined) return this.props.children;

    return (
      <div className="flex flex-col items-start gap-2 rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
        <p className="text-xs text-bad">This screen could not be drawn: {this.state.said}</p>
        <p className="text-[11px] text-faint">
          Nothing is lost — it is a fault in the page, not in the record. The console has the
          whole of it.
        </p>
        <div className="flex gap-1.5">
          <button
            onClick={() => this.setState({ said: undefined })}
            className="rounded border border-edge px-2 py-0.5 text-[11px] text-said hover:bg-hover"
          >
            Draw it again
          </button>
          {this.props.back && (
            <button
              onClick={() => {
                this.setState({ said: undefined });
                this.props.back?.();
              }}
              className="rounded border border-edge px-2 py-0.5 text-[11px] text-aside hover:bg-hover"
            >
              Back to the board
            </button>
          )}
        </div>
      </div>
    );
  }
}
