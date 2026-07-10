import EntryCatalog from "../components/catalog/EntryCatalog";
import { usePageTitle } from "../hooks/usePageTitle";

export default function Browse() {
  usePageTitle("Browse");
  return <EntryCatalog mode="browse" />;
}
