/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type FetchStatsDetailed = {
    /**
     * CDC cursor
     */
    cdc_cursor: string;
    /**
     * export duration in milliseconds
     */
    export_duration_ms: number;
    /**
     * import duration in milliseconds
     */
    import_duration_ms: number;
    /**
     * net duration in milliseconds
     */
    net_duration_ms: number;
    /**
     * number of errors processed
     */
    num_errors: number;
    /**
     * number of rows
     */
    num_rows: number;
    /**
     * number of tables processed
     */
    num_tables: number;
    /**
     * percentage complete of fetch run
     */
    percent_complete: string;
};

