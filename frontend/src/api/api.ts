// frontend/src/api/api.ts
import * as API from '../../wailsjs/go/spgui/SPGUI';

// ===== Types (espelham os DTOs do backend) =====
export type Config = {
  ConfigPath: string;
  SiteURL: string;
  GlobalTimeoutSec: number; // 0 = sem limite
  CleanOutput: boolean;
};

export type ListQuery = {
  list: string;
  select: string;
  filter: string;
  orderby: string;
  top: number;
  all: boolean;
  latestOnly: boolean;
};

export type QuerySummary = {
  items: number;
  pages: number;           // PagesFetched (json:"pages")
  throttled: boolean;
  partial: boolean;
  fallback: boolean;       // UsedFallback (json:"fallback")
  stoppedEarly: boolean;
};

export type ListResponse = {
  items: Record<string, any>[];
  summary: QuerySummary;
};

export type AttachmentInfo = {
  fileName: string;
  serverRelativeUrl: string;
};

// ===== API tipada =====
async function ping() {
  return API.Ping();
}

async function setConfig(cfg: Config) {
  return API.SetConfig(cfg);
}

async function cancelCurrent() {
  return API.CancelCurrent();
}

async function listItems(q: ListQuery): Promise<ListResponse> {
  const res = await API.ListItems(q);
  return res as unknown as ListResponse;
}

async function getItem(list: string, id: number, selectFields: string) {
  return API.GetItem(list, id, selectFields) as Promise<Record<string, any>>;
}

async function addItem(list: string, fields: Record<string, any>, selectFields: string) {
  return API.AddItem(list, fields, selectFields) as Promise<Record<string, any>>;
}

async function updateItem(list: string, id: number, fields: Record<string, any>, selectFields: string) {
  return API.UpdateItem(list, id, fields, selectFields) as Promise<Record<string, any>>;
}

async function deleteItem(list: string, id: number) {
  return API.DeleteItem(list, id); // boolean
}

// --- Attachments ---
async function listAttachments(list: string, id: number): Promise<AttachmentInfo[]> {
  const out = await API.ListAttachments(list, id);
  return out as unknown as AttachmentInfo[];
}

async function addAttachment(list: string, id: number, fileName: string, content: number[]) {
  const info = await API.AddAttachment(list, id, fileName, content);
  return info as unknown as AttachmentInfo;
}

async function downloadAttachment(list: string, id: number, fileName: string) {
  return API.DownloadAttachment(list, id, fileName) as Promise<number[] | string>;
}

async function deleteAttachment(list: string, id: number, fileName: string) {
  return API.DeleteAttachment(list, id, fileName); // boolean
}

async function downloadByURL(urlOrPath: string) {
  return API.DownloadByURL(urlOrPath) as Promise<number[] | string>;
}

// --- Variantes com SAVE DIALOG (novas) ---
async function saveAttachmentPick(list: string, id: number, fileName: string) {
  return API.SaveAttachmentPick(list, id, fileName) as Promise<string>; // path ou ""
}

async function saveByURLPick(urlOrPath: string) {
  return API.SaveByURLPick(urlOrPath) as Promise<string>; // path ou ""
}

async function saveBytesPick(defaultFilename: string, content: number[], mime?: string) {
  return API.SaveBytesPick(defaultFilename, content, mime ?? '') as Promise<string>; // path ou ""
}

// --- Diálogo nativo (OpenConfigDialog) ---
async function openConfigDialog(): Promise<string> {
  const p = await API.OpenConfigDialog();
  return (p ?? '') as string;
}

export const sp = {
  ping,
  setConfig,
  cancelCurrent,
  listItems,
  getItem,
  addItem,
  updateItem,
  deleteItem,
  listAttachments,
  addAttachment,
  downloadAttachment,
  deleteAttachment,
  downloadByURL,
  saveAttachmentPick,
  saveByURLPick,
  saveBytesPick,
  openConfigDialog,
};

export default sp;
