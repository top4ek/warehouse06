import EntryCatalog from "../components/catalog/EntryCatalog";
import { usePageTitle } from "../hooks/usePageTitle";

export default function Authors() {
  usePageTitle("Authors");
  return <EntryCatalog mode="authors" />;
}
