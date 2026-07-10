import { useCallback, useState } from "react";

type OpenChangeInfo = {
  source: "trigger" | "menu" | "outside";
};

/** Keeps the dropdown open when menu items are clicked; closes on outside click or trigger toggle. */
export function useHeaderDropdown() {
  const [open, setOpen] = useState(false);

  const onOpenChange = useCallback((nextOpen: boolean, info: OpenChangeInfo) => {
    if (info.source === "menu") return;
    setOpen(nextOpen);
  }, []);

  return { open, onOpenChange };
}
