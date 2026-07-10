import { CloseOutlined } from "@ant-design/icons";
import { Button, Flex, Typography, theme } from "antd";
import {
  useEffect,
  useRef,
  type CSSProperties,
  type ReactNode,
  type Ref,
} from "react";
import { createPortal } from "react-dom";

export type UiWindowProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  titleCenter?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  toolbar?: ReactNode;
  ariaLabel?: string;
  width?: number | string;
  maxWidth?: string;
  fullscreen?: boolean;
  closeOnBackdrop?: boolean;
  closeOnEscape?: boolean;
  showClose?: boolean;
  windowRef?: Ref<HTMLDivElement>;
  className?: string;
  bodyClassName?: string;
};

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (!ref) return;
  if (typeof ref === "function") ref(value);
  else ref.current = value;
}

function portalContainer(): HTMLElement {
  return document.querySelector<HTMLElement>(".ant-app") ?? document.body;
}

export default function UiWindow({
  open,
  onClose,
  title,
  titleCenter,
  children,
  footer,
  toolbar,
  ariaLabel,
  width,
  maxWidth,
  fullscreen = false,
  closeOnBackdrop = true,
  closeOnEscape = true,
  showClose = true,
  windowRef,
  className,
  bodyClassName,
}: UiWindowProps) {
  const { token } = theme.useToken();
  const backdropRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  useEffect(() => {
    if (!open || !closeOnEscape) return;
    function onKeydown(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKeydown);
    return () => window.removeEventListener("keydown", onKeydown);
  }, [open, closeOnEscape, onClose]);

  if (!open) return null;

  const sizeStyle: CSSProperties = fullscreen
    ? { width: "100%", maxWidth: "100%", height: "100%", maxHeight: "100%" }
    : {
        width: width ?? "100%",
        maxWidth: maxWidth ?? "min(100vw - 32px, 960px)",
      };

  const windowStyle: CSSProperties = {
    ...sizeStyle,
    background: token.colorBgContainer,
    color: token.colorText,
    border: fullscreen ? "none" : `1px solid ${token.colorBorderSecondary}`,
    borderRadius: fullscreen ? 0 : token.borderRadiusLG,
    boxShadow: fullscreen ? "none" : token.boxShadowSecondary,
  };

  const titleLabel = typeof title === "string" ? title : ariaLabel;

  return createPortal(
    <div
      ref={backdropRef}
      className={`ui-window-backdrop${fullscreen ? " ui-window-backdrop--fullscreen" : ""}`}
      style={{ background: token.colorBgMask }}
      role="presentation"
      onClick={(event) => {
        if (closeOnBackdrop && event.target === backdropRef.current) onClose();
      }}
    >
      <div
        ref={(node) => assignRef(windowRef, node)}
        className={["ui-window", fullscreen && "ui-window--fullscreen", className]
          .filter(Boolean)
          .join(" ")}
        style={windowStyle}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel ?? titleLabel}
        onClick={(event) => event.stopPropagation()}
      >
        <div
          className={[
            "ui-window__titlebar",
            titleCenter && "ui-window__titlebar--with-center",
          ]
            .filter(Boolean)
            .join(" ")}
          style={{
            padding: `${token.paddingXS}px ${token.paddingMD}px`,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            background: token.colorBgContainer,
          }}
        >
          <Typography.Text ellipsis className="ui-window__titlebar-start">
            {title}
          </Typography.Text>
          {titleCenter ? (
            <div className="ui-window__titlebar-center">{titleCenter}</div>
          ) : null}
          <Flex gap={token.marginXXS} align="center" className="ui-window__titlebar-end">
            {toolbar}
            {showClose && (
              <Button
                type="text"
                size="small"
                icon={<CloseOutlined />}
                onClick={onClose}
                aria-label="Close"
              />
            )}
          </Flex>
        </div>
        <div
          className={["ui-window__body", bodyClassName].filter(Boolean).join(" ")}
          style={{ background: token.colorBgContainer }}
        >
          {children}
        </div>
        {footer ? (
          <div
            className="ui-window__footer"
            style={{
              padding: `${token.paddingXS}px ${token.paddingMD}px`,
              borderTop: `1px solid ${token.colorBorderSecondary}`,
              background: token.colorBgContainer,
              color: token.colorTextSecondary,
            }}
          >
            {footer}
          </div>
        ) : null}
      </div>
    </div>,
    portalContainer(),
  );
}
