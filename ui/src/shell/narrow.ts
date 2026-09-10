// Whether the screen is a hand or a desk.
//
// Tailwind's breakpoints answer this in CSS, and most of the layout is
// better off saying it there. This is for the few places where the shape is
// not a class but a decision: a split diff at 390 columns is two columns of
// forty characters, which is not a diff — so the component has to choose
// unified, not just hide the switch.

import { useEffect, useState } from "react";

// desk is Tailwind's lg, where the panes stop competing for width.
const desk = "(min-width: 1024px)";

export function useDesk(): boolean {
  const [wide, setWide] = useState(() => window.matchMedia(desk).matches);

  useEffect(() => {
    const query = window.matchMedia(desk);
    const onChange = () => setWide(query.matches);

    query.addEventListener("change", onChange);

    return () => query.removeEventListener("change", onChange);
  }, []);

  return wide;
}
