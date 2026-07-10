import { useThemeMode } from "../context/ThemeContext";

const MARKS = {
  light: "/logo-mark.svg",
  dark: "/logo-mark-dark.svg",
} as const;

type BrandMarkProps = {
  className?: string;
  width?: number;
  height?: number;
};

export default function BrandMark({ className, width = 32, height = 32 }: BrandMarkProps) {
  const { mode } = useThemeMode();

  return (
    <img
      src={MARKS[mode]}
      alt=""
      width={width}
      height={height}
      className={className}
      decoding="async"
    />
  );
}
