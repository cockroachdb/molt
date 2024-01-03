/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type CreateFetchTaskRequestBody = {
    /**
     * the local address CRDB will use to access the import files
     */
    bucket_name: string;
    /**
     * the sub-path within the bucket to write the export files
     */
    bucket_path: string;
    /**
     * whether the intermediate store should be cleaned up after the fetch task
     */
    cleanup_intermediary_store: boolean;
    /**
     * compression type
     */
    compression: CreateFetchTaskRequestBody.compression;
    /**
     * the absolute or relative path to write export files
     */
    local_path: string;
    /**
     * the local address CRDB will use to access the import files
     */
    local_path_crdb_address: string;
    /**
     * the local address where the file server will be spun up
     */
    local_path_listen_address: string;
    /**
     * if specified, writes task execution logs to a file and stdout; otherwise, just writes to stdout
     */
    log_file: string;
    /**
     * Mode of operation for fetch
     */
    mode: CreateFetchTaskRequestBody.mode;
    /**
     * the name of the fetch run
     */
    name: string;
    /**
     * number of rows to export at a given time for each iteration; tune this so that you get most out of CPU and can batch the most data together
     */
    num_batch_rows_export: number;
    /**
     * number of tables to process at the same time; this is usually sized based on number of CPUs
     */
    num_concurrent_tables: number;
    /**
     * number of bytes for the export before data is flushed to the disk
     */
    num_flush_bytes: number;
    /**
     * number of rows for the export before data is flushed to the disk (persisted)
     */
    num_flush_rows: number;
    /**
     * if set and exists, drops the existing replication slot
     */
    pg_drop_slot: boolean;
    /**
     * name for pg replication plugin
     */
    pg_logical_plugin: string;
    /**
     * name for pg replication slot
     */
    pg_logical_slot_name: string;
    /**
     * source database connection string
     */
    source_conn: string;
    /**
     * Type of intermediary store
     */
    store: CreateFetchTaskRequestBody.store;
    /**
     * target database connection string
     */
    target_conn: string;
    /**
     * if specified, truncates the target tables before running the data load
     */
    truncate: boolean;
};

export namespace CreateFetchTaskRequestBody {

    /**
     * compression type
     */
    export enum compression {
        GZIP = 'gzip',
        NONE = 'none',
    }

    /**
     * Mode of operation for fetch
     */
    export enum mode {
        IMPORT_INTO = 'IMPORT_INTO',
        COPY_FROM = 'COPY_FROM',
        DIRECT_COPY = 'DIRECT_COPY',
    }

    /**
     * Type of intermediary store
     */
    export enum store {
        NONE = 'None',
        AWS = 'AWS',
        GCP = 'GCP',
        LOCAL = 'Local',
    }


}

