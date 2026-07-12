import { SearchOutlined } from "@ant-design/icons";
import { Button, Input } from "antd";
import type { InputRef } from "antd";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { browsePath } from "../lib/browse";

export default function SearchBar() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const urlQuery = searchParams.get("q") ?? "";
  const [query, setQuery] = useState(urlQuery);
  const [expanded, setExpanded] = useState(!!urlQuery);
  const isMobile = useMediaQuery("(max-width: 767px)");
  // Collapsing to an icon only happens on mobile; on desktop the bar is always open.
  const isOpen = expanded || !isMobile;
  const inputRef = useRef<InputRef>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  // Sync the input with the ?q= URL param (adjust-during-render,
  // https://react.dev/learn/you-might-not-need-an-effect).
  const [lastUrlQuery, setLastUrlQuery] = useState(urlQuery);
  if (urlQuery !== lastUrlQuery) {
    setLastUrlQuery(urlQuery);
    setQuery(urlQuery);
    if (urlQuery) setExpanded(true);
  }

  useEffect(() => {
    if (!expanded) return;
    const id = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(id);
  }, [expanded]);

  useEffect(() => {
    if (!expanded) return;
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node) && !query.trim()) {
        setExpanded(false);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [expanded, query]);

  function submit(value?: string) {
    const trimmed = (value ?? query).trim();
    if (!trimmed) return;
    navigate(browsePath({ q: trimmed }));
  }

  function onTriggerClick() {
    if (!isOpen) {
      setExpanded(true);
      return;
    }
    if (query.trim()) submit();
    else inputRef.current?.focus();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") submit();
    if (e.key === "Escape" && !query.trim()) setExpanded(false);
  }

  return (
    <div
      ref={rootRef}
      className={`search-bar${isOpen ? " search-bar--expanded" : ""}`}
      role="search"
    >
      <Button
        type="text"
        icon={<SearchOutlined />}
        aria-label="Search"
        aria-expanded={isOpen}
        onClick={onTriggerClick}
        className="search-bar__trigger"
      />
      <Input
        ref={inputRef}
        id="global-search"
        type="search"
        size="middle"
        placeholder="Search entries, authors, tags..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={onKeyDown}
        onPressEnter={() => submit()}
        allowClear
        className="search-bar__input"
        aria-label="Search query"
      />
    </div>
  );
}
