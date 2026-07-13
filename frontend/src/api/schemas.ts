import { z } from "zod";

export const tagSchema = z.object({
  id: z.number(),
  name: z.string(),
  entry_count: z.number().optional(),
});

export const authorSchema = z.object({
  id: z.number(),
  directory_name: z.string(),
  name: z.string(),
  address: z.string().optional(),
  content_html: z.string().optional(),
  preview_image: z.string().optional(),
  entry_count: z.number().optional(),
});

export const fileSchema = z.object({
  id: z.number(),
  filename: z.string(),
  filepath: z.string(),
  is_image: z.boolean(),
});

export const directorySchema = z.object({
  name: z.string(),
  path: z.string(),
});

const controlCellSchema = z.union([
  z.null(),
  z.string(),
  z.object({ key: z.string(), label: z.string().optional() }),
]);

export const controlsSchema = z.object({
  rows: z.array(z.array(controlCellSchema)),
});

export const entrySchema = z.object({
  id: z.number(),
  path: z.string(),
  name: z.string(),
  description: z.string(),
  description_html: z.string().optional(),
  content_html: z.string().optional(),
  date: z.string().optional(),
  created_at: z.string().optional(),
  type: z.string().optional(),
  youtube: z.string().optional(),
  platform: z.string().optional(),
  preview_image: z.string().optional(),
  entry_count: z.number().optional(),
  tags: z.array(tagSchema).optional(),
  authors: z.array(authorSchema).optional(),
  files: z.array(fileSchema).optional(),
  directories: z.array(directorySchema).optional(),
  require: z.array(z.string()).optional(),
  // A malformed controls frontmatter must not invalidate the whole entry.
  controls: controlsSchema.nullish().catch(null),
});

export const entryListResultSchema = z.object({
  items: z.array(entrySchema),
  total: z.number(),
});

export const authorListSchema = z.array(authorSchema);

export const tagListSchema = z.array(tagSchema);

export const platformSchema = z.object({
  path: z.string(),
  name: z.string(),
  description: z.string().optional(),
  content_html: z.string().optional(),
  entry_count: z.number(),
});

export const platformListSchema = z.array(platformSchema);

export const storageCommitSchema = z.object({
  hash: z.string(),
  committed_at: z.string(),
  subject: z.string().optional(),
});

export const syncStatusSchema = z.object({
  syncing: z.boolean(),
  last_synced_at: z.string().optional(),
  storage_commit: storageCommitSchema.optional(),
});

export type ControlsConfig = z.infer<typeof controlsSchema>;
export type Tag = z.infer<typeof tagSchema>;
export type Author = z.infer<typeof authorSchema>;
export type FileItem = z.infer<typeof fileSchema>;
export type DirectoryItem = z.infer<typeof directorySchema>;
export type Entry = z.infer<typeof entrySchema>;
export type Platform = z.infer<typeof platformSchema>;
export type EntryListResult = z.infer<typeof entryListResultSchema>;
export type StorageCommit = z.infer<typeof storageCommitSchema>;
export type SyncStatus = z.infer<typeof syncStatusSchema>;
