import { useCallback, useLayoutEffect, useRef, useState } from "react";
import { Link as RouterLink } from "react-router-dom";

export type NavItem = {
  key: string;
  label: string;
  to: string;
};

type Indicator = {
  left: number;
  width: number;
  ready: boolean;
};

type AppNavMenuProps = {
  items: NavItem[];
  activeKey: string;
  className?: string;
};

export default function AppNavMenu({ items, activeKey, className }: AppNavMenuProps) {
  const navRef = useRef<HTMLElement>(null);
  const itemRefs = useRef(new Map<string, HTMLAnchorElement>());
  const [indicator, setIndicator] = useState<Indicator>({ left: 0, width: 0, ready: false });

  const updateIndicator = useCallback(() => {
    const nav = navRef.current;
    const activeEl = itemRefs.current.get(activeKey);
    if (!nav || !activeEl) {
      setIndicator((prev) => ({ ...prev, ready: false }));
      return;
    }

    const navRect = nav.getBoundingClientRect();
    const itemRect = activeEl.getBoundingClientRect();
    setIndicator({
      left: itemRect.left - navRect.left + nav.scrollLeft,
      width: itemRect.width,
      ready: true,
    });
  }, [activeKey]);

  useLayoutEffect(() => {
    updateIndicator();
  }, [updateIndicator, items]);

  useLayoutEffect(() => {
    const nav = navRef.current;
    if (!nav) return undefined;

    const observer = new ResizeObserver(() => updateIndicator());
    observer.observe(nav);
    window.addEventListener("resize", updateIndicator);
    nav.addEventListener("scroll", updateIndicator, { passive: true });

    return () => {
      observer.disconnect();
      window.removeEventListener("resize", updateIndicator);
      nav.removeEventListener("scroll", updateIndicator);
    };
  }, [updateIndicator]);

  const navClassName = className ? `app-nav ${className}` : "app-nav";

  return (
    <nav ref={navRef} className={navClassName} aria-label="Main">
      <span
        className="app-nav__indicator"
        aria-hidden
        data-ready={indicator.ready || undefined}
        style={{
          transform: `translate3d(${indicator.left}px, 0, 0)`,
          width: indicator.width,
        }}
      />
      {items.map((item) => {
        const active = item.key === activeKey;
        return (
          <RouterLink
            key={item.key}
            ref={(el) => {
              if (el) itemRefs.current.set(item.key, el);
              else itemRefs.current.delete(item.key);
            }}
            to={item.to}
            className={`app-nav__item${active ? " app-nav__item--active" : ""}`}
            aria-current={active ? "page" : undefined}
          >
            {item.label}
          </RouterLink>
        );
      })}
    </nav>
  );
}
