import { Alert, Button, Flex } from "antd";
import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = { children: ReactNode };
type State = { error: Error | null };

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("UI error:", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      const staleChunk =
        this.state.error.message.includes("dynamically imported module") ||
        this.state.error.message.includes("Importing a module script failed");
      return (
        <Flex justify="center" style={{ padding: 48 }}>
          <Alert
            type="error"
            showIcon
            message="Something went wrong"
            description={
              staleChunk
                ? `${this.state.error.message} — likely an outdated cached script. Reload the page (Ctrl+Shift+R).`
                : this.state.error.message
            }
            action={
              <Flex gap={8}>
                {staleChunk && (
                  <Button
                    size="small"
                    type="primary"
                    onClick={() => window.location.reload()}
                  >
                    Reload app
                  </Button>
                )}
                <Button size="small" onClick={() => this.setState({ error: null })}>
                  Try again
                </Button>
              </Flex>
            }
          />
        </Flex>
      );
    }
    return this.props.children;
  }
}
