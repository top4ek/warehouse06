import {
  ControlOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  QuestionCircleOutlined,
} from "@ant-design/icons";
import { Button, theme } from "antd";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import UiWindow from "./common/UiWindow";
import EmulatorControls from "./EmulatorControls";
import type { ControlsConfig } from "../api/types";
import { emulatorFrameSrc } from "../lib/emulator";
import { hostKeyCode, postEmulatorKey } from "../lib/emulatorInput";
import { useMediaQuery } from "../hooks/useMediaQuery";

const hotkeys = [
  { key: "F11", desc: "Перезагрузка (БЛК+ВВОД)" },
  { key: "F12", desc: "Рестарт / запуск диска (БЛК+СБР)" },
  { key: "F6", desc: "РУС / ЛАТ" },
  { key: "Shift", desc: "СС" },
  { key: "Ctrl", desc: "УС" },
  { key: "Alt", desc: "ПС" },
  { key: "Shift+Bksp", desc: "СТР" },
];

const TITLEBAR_CHROME = 41;

type ScreenSize = { width: number; height: number };

function fitScreen4x3(viewportW: number, viewportH: number, margin: number, chrome: number): ScreenSize {
  const maxW = Math.max(0, viewportW - margin * 2);
  const maxH = Math.max(0, viewportH - margin * 2 - chrome);
  let width = maxW;
  let height = (width * 3) / 4;
  if (height > maxH) {
    height = maxH;
    width = (height * 4) / 3;
  }
  return { width: Math.floor(width), height: Math.floor(height) };
}

type Props = {
  open: boolean;
  romUrl: string;
  onClose: () => void;
  controls?: ControlsConfig | null;
};

export default function EmulatorDialog({ open, romUrl, onClose, controls }: Props) {
  const { token } = theme.useToken();
  const isMobile = useMediaQuery("(max-width: 767px)");
  const [showHelp, setShowHelp] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [controlsVisible, setControlsVisible] = useState(isMobile);
  const [screenSize, setScreenSize] = useState<ScreenSize>({ width: 960, height: 720 });
  const windowRef = useRef<HTMLDivElement>(null);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const helpRef = useRef<HTMLDivElement>(null);
  const controlsRef = useRef<HTMLDivElement>(null);
  const viewportMargin = token.marginXL;

  const frameSrc = useMemo(() => {
    if (!open || !romUrl) return "about:blank";
    return emulatorFrameSrc(romUrl);
  }, [open, romUrl]);

  // Reset transient UI state when the dialog closes (adjust-during-render,
  // https://react.dev/learn/you-might-not-need-an-effect).
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setControlsVisible(isMobile);
    } else {
      setShowHelp(false);
      setFullscreen(false);
    }
  }

  useLayoutEffect(() => {
    if (!open || fullscreen) return;

    function update() {
      const helpH = showHelp ? (helpRef.current?.offsetHeight ?? 0) : 0;
      const controlsH = controlsVisible ? (controlsRef.current?.offsetHeight ?? 0) : 0;
      setScreenSize(
        fitScreen4x3(
          window.innerWidth,
          window.innerHeight,
          viewportMargin,
          TITLEBAR_CHROME + helpH + controlsH,
        ),
      );
    }

    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, [open, fullscreen, showHelp, controlsVisible, viewportMargin]);

  useEffect(() => {
    function onFullscreenChange() {
      setFullscreen(document.fullscreenElement === windowRef.current);
    }
    document.addEventListener("fullscreenchange", onFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", onFullscreenChange);
  }, []);

  useEffect(() => {
    if (!open) return;

    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur();
    }

    function isBlockedTarget(target: EventTarget | null): boolean {
      if (!(target instanceof HTMLElement)) return false;
      const tag = target.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable) return true;
      return !!target.closest(".ui-window__titlebar button, .ui-window__titlebar a");
    }

    function forwardKey(event: KeyboardEvent, subcmd: "keydown" | "keyup") {
      if (isBlockedTarget(event.target)) return;
      const keycode = hostKeyCode(event);
      if (!keycode) return;
      postEmulatorKey(frameRef.current, subcmd, keycode);
      event.preventDefault();
      event.stopPropagation();
    }

    function onKeydown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        if (document.fullscreenElement === windowRef.current) {
          void document.exitFullscreen().catch(() => {});
        } else {
          onClose();
        }
        return;
      }
      forwardKey(event, "keydown");
    }

    function onKeyup(event: KeyboardEvent) {
      if (event.key === "Escape") return;
      forwardKey(event, "keyup");
    }

    document.addEventListener("keydown", onKeydown, true);
    document.addEventListener("keyup", onKeyup, true);
    return () => {
      document.removeEventListener("keydown", onKeydown, true);
      document.removeEventListener("keyup", onKeyup, true);
    };
    // The handlers read frameRef.current live, so a frameSrc change must not
    // tear the listeners down.
  }, [open, onClose]);

  async function toggleFullscreen() {
    const el = windowRef.current;
    if (!el || !document.fullscreenEnabled) return;
    if (document.fullscreenElement === el) {
      await document.exitFullscreen().catch(() => {});
    } else {
      await el.requestFullscreen();
    }
  }

  function handleClose() {
    if (document.fullscreenElement === windowRef.current) {
      void document.exitFullscreen().catch(() => {});
    }
    onClose();
  }

  return (
    <UiWindow
      open={open}
      onClose={handleClose}
      title="Vector-06C Emulator"
      titleCenter={
        !fullscreen ? (
          <p className="ui-window__emulator-credit">
            Powered by{" "}
            <a href="https://github.com/svofski/vector06js" target="_blank" rel="noreferrer">
              vector06js
            </a>
          </p>
        ) : undefined
      }
      ariaLabel="Vector-06C Emulator"
      width={screenSize.width}
      maxWidth={fullscreen ? "100%" : `calc(100vw - ${viewportMargin * 2}px)`}
      fullscreen={fullscreen}
      closeOnEscape={false}
      closeOnBackdrop={!fullscreen}
      showClose={false}
      windowRef={windowRef}
      className="ui-window--emulator"
      bodyClassName="ui-window__body--flush"
      toolbar={
        <>
          <Button
            type="text"
            size="small"
            icon={<QuestionCircleOutlined />}
            onClick={() => setShowHelp((v) => !v)}
            aria-expanded={showHelp}
            aria-label="Hotkeys"
          />
          <Button
            type="text"
            size="small"
            icon={<ControlOutlined />}
            onClick={() => setControlsVisible((v) => !v)}
            aria-pressed={controlsVisible}
            aria-label="On-screen controls"
          />
          <Button
            type="text"
            size="small"
            icon={fullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            onClick={() => void toggleFullscreen()}
            aria-label="Fullscreen"
          />
          <Button type="text" size="small" onClick={handleClose} aria-label="Close">
            Close
          </Button>
        </>
      }
    >
      {showHelp && (
        <div
          ref={helpRef}
          className="emulator-help"
          style={{
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            background: token.colorBgContainer,
          }}
        >
          <div className="emulator-help__list">
            {hotkeys.map(({ key, desc }) => (
              <span key={key} className="emulator-help__item">
                <span className="emulator-help__key" style={{ color: token.colorPrimary }}>
                  {key}
                </span>
                <span className="emulator-help__desc">{desc}</span>
              </span>
            ))}
          </div>
        </div>
      )}

      {open && (
        <iframe
          ref={frameRef}
          key={frameSrc}
          src={frameSrc}
          title="Vector-06C Emulator"
          allow="autoplay; keyboard; gamepad; fullscreen"
          className="ui-window__emulator-frame"
          style={{
            width: "100%",
            // In fullscreen with the controls panel below, the frame is sized
            // by flex (see .ui-window--fullscreen rules) instead of 100%,
            // which would overflow by the panel height.
            height: fullscreen ? (controlsVisible ? undefined : "100%") : screenSize.height,
            border: 0,
            display: "block",
          }}
        />
      )}

      {open && controlsVisible && (
        <div ref={controlsRef}>
          <EmulatorControls
            controls={controls}
            onKey={(subcmd, keycode) => postEmulatorKey(frameRef.current, subcmd, keycode)}
          />
        </div>
      )}
    </UiWindow>
  );
}
