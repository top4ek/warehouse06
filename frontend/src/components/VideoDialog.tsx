import { parseYouTubeId } from "../lib/youtube";
import UiWindow from "./common/UiWindow";

type Props = {
  open: boolean;
  youtubeId: string;
  onClose: () => void;
};

export default function VideoDialog({ open, youtubeId, onClose }: Props) {
  const safeId = parseYouTubeId(youtubeId);
  return (
    <UiWindow
      open={open && Boolean(safeId)}
      onClose={onClose}
      title="Video"
      ariaLabel="Video"
      width="100%"
      maxWidth="800px"
      bodyClassName="ui-window__body--flush"
    >
      {safeId && (
        <iframe
          src={`https://www.youtube.com/embed/${safeId}?autoplay=1`}
          title="Video"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          style={{ width: "100%", aspectRatio: "16/9", border: 0, display: "block" }}
        />
      )}
    </UiWindow>
  );
}
