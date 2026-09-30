import { useLayoutEffect, useRef } from "react";

const stack: HTMLElement[] = [];
const inertBefore = new Map<HTMLElement, boolean>();
let previousOverflow = "";
const focusable =
  'a[href], button, input, select, textarea, [tabindex], [contenteditable="true"]';

function updateBackground() {
  const top = stack.at(-1);
  for (const [element, inert] of inertBefore) element.inert = inert;
  inertBefore.clear();
  if (!top) return;
  for (const element of document.body.children) {
    if (element instanceof HTMLElement && !element.contains(top)) {
      inertBefore.set(element, element.inert);
      element.inert = true;
    }
  }
}

export function useDialog(
  open: boolean,
  onClose: () => void,
  initialFocus?: React.RefObject<HTMLElement | null>,
) {
  const ref = useRef<HTMLDivElement>(null);
  const closeRef = useRef(onClose);
  useLayoutEffect(() => {
    closeRef.current = onClose;
  });

  useLayoutEffect(() => {
    const dialog = ref.current;
    if (!open || !dialog) return;
    const previousFocus =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    if (stack.length === 0) {
      previousOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
    }
    stack.push(dialog);
    const getFocusable = () =>
      Array.from(dialog.querySelectorAll<HTMLElement>(focusable)).filter(
        (element) =>
          element.tabIndex >= 0 &&
          !element.matches(":disabled") &&
          !element.closest("[inert]") &&
          element.getClientRects().length > 0,
      );
    const focusFirst = () =>
      (getFocusable()[0] ?? dialog).focus({ preventScroll: true });
    updateBackground();
    (initialFocus?.current ?? getFocusable()[0] ?? dialog).focus({
      preventScroll: true,
    });
    const observer = new MutationObserver(updateBackground);
    observer.observe(document.body, { childList: true });
    const keydown = (event: KeyboardEvent) => {
      if (stack.at(-1) !== dialog) return;
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopImmediatePropagation();
        closeRef.current();
      } else if (event.key === "Tab") {
        const elements = getFocusable();
        const index = elements.indexOf(document.activeElement as HTMLElement);
        if (
          elements.length === 0 ||
          index < 0 ||
          (!event.shiftKey && index === elements.length - 1) ||
          (event.shiftKey && index === 0)
        ) {
          event.preventDefault();
          (event.shiftKey
            ? (elements.at(-1) ?? dialog)
            : (elements[0] ?? dialog)
          ).focus();
        }
      }
    };
    const focusin = (event: FocusEvent) => {
      if (stack.at(-1) === dialog && !dialog.contains(event.target as Node))
        focusFirst();
    };
    document.addEventListener("keydown", keydown, true);
    document.addEventListener("focusin", focusin);
    return () => {
      observer.disconnect();
      document.removeEventListener("keydown", keydown, true);
      document.removeEventListener("focusin", focusin);
      stack.splice(stack.indexOf(dialog), 1);
      updateBackground();
      if (stack.length === 0) document.body.style.overflow = previousOverflow;
      if (previousFocus?.isConnected && !previousFocus.closest("[inert]"))
        previousFocus.focus({ preventScroll: true });
    };
  }, [open, initialFocus]);
  return ref;
}
