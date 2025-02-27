import { InstanceId } from "./id";

export type QueryInfo = {
  instanceId: InstanceId;
  databaseName?: string;
  statement: string;
  limit?: number;
  // exportFomat includes QUERY, CSV, JSON.
  exportFormat?: string;
};
