import { CreateFetchTaskRequestBody, MoltserviceService } from "../apigen";
import { CompressionType, IntermediateStore, Mode, TaskFormState } from "../pages/ConfigureTask";

// TODO: configure the OpenAPI config based on environment

export const getFetchTasks = () => {
    return MoltserviceService.moltserviceGetFetchTasks()
}

export const getSpecificFetchTask = (id: number) => {
    return MoltserviceService.moltserviceGetSpecificFetchTask(id)
}

const getCompressionFromType = (compression: CompressionType): CreateFetchTaskRequestBody.compression => {
    switch (compression) {
        case "gzip":
            return CreateFetchTaskRequestBody.compression.GZIP
        case "none":
            return CreateFetchTaskRequestBody.compression.NONE
    }

    return CreateFetchTaskRequestBody.compression.DEFAULT
}

const getModeFromType = (mode: Mode): CreateFetchTaskRequestBody.mode => {
    switch (mode) {
        case "directCopy":
            return CreateFetchTaskRequestBody.mode.DIRECT_COPY
        case "liveCopyFromStore":
            return CreateFetchTaskRequestBody.mode.COPY_FROM
    }

    return CreateFetchTaskRequestBody.mode.IMPORT_INTO
}

const getStoreFromType = (store: IntermediateStore): CreateFetchTaskRequestBody.store => {
    switch (store) {
        case "local":
            return CreateFetchTaskRequestBody.store.LOCAL
        case "S3":
            return CreateFetchTaskRequestBody.store.AWS
        case "GCS":
            return CreateFetchTaskRequestBody.store.GCP
    }

    return CreateFetchTaskRequestBody.store.NONE
}

export const createVerifyFromFetchTask = (fetchId: number) => {
    return MoltserviceService.moltserviceCreateVerifyTaskFromFetch(fetchId);
}

export const createFetchTask = (task: TaskFormState) => {
    const reqBody: CreateFetchTaskRequestBody = {
        bucket_name: task.bucketName,
        bucket_path: task.bucketPath,
        cleanup_intermediary_store: task.cleanup,
        compression: getCompressionFromType(task.compression),
        local_path: task.localPath,
        local_path_crdb_address: task.localPathCRDBAccessAddr,
        local_path_listen_address: task.localPathListenAddr,
        log_file: task.logFile,
        mode: getModeFromType(task.mode),
        name: task.name,
        num_batch_rows_export: task.numBatchRowsExport,
        num_concurrent_tables: task.numConcurrentTables,
        num_flush_bytes: task.flushSize,
        num_flush_rows: task.flushNumRows,
        pg_drop_slot: task.dropPgLogicalSlot,
        pg_logical_plugin: task.pgLogicalSlotPlugin,
        pg_logical_slot_name: task.pgLogicalSlotName,
        source_conn: task.sourceURL,
        store: getStoreFromType(task.store),
        target_conn: task.targetURL,
        truncate: task.truncate
    }

    return MoltserviceService.moltserviceCreateFetchTask(reqBody);
}
