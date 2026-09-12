// A price, written the way money is written.
//
// Four decimals everywhere drew forty-two cents as $0.4200, which is not a
// number anybody writes down. Two decimals is money — except when the
// amount is smaller than a cent, where two would round a run that cost
// something to nothing at all, and a run that spent must never be drawn as
// free.
export function money(n: number): string {
  return n > 0 && n < 0.01 ? `$${n.toFixed(4)}` : `$${n.toFixed(2)}`;
}
