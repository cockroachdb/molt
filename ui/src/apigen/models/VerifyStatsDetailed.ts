/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type VerifyStatsDetailed = {
    /**
     * net duration in milliseconds
     */
    net_duration_ms: number;
    /**
     * number column mismatches
     */
    num_column_mismatch: number;
    /**
     * number of rows that had conditional success
     */
    num_conditional_success: number;
    /**
     * number of extraneous rows
     */
    num_extraneous: number;
    /**
     * number of live retries
     */
    num_live_retry: number;
    /**
     * number of mismatching rows
     */
    num_mismatch: number;
    /**
     * number of missing rows
     */
    num_missing: number;
    /**
     * number of successful rows processed
     */
    num_success: number;
    /**
     * number of tables processed
     */
    num_tables: number;
    /**
     * number of rows processed
     */
    num_truth_rows: number;
};

