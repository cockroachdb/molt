/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type VerifyRun = {
    /**
     * ID of the associated fetch run
     */
    fetch_id: number;
    /**
     * finished at time
     */
    finished_at: number;
    /**
     * ID of the run
     */
    id: number;
    /**
     * name of the run
     */
    name: string;
    /**
     * started at time
     */
    started_at: number;
    /**
     * status of the run
     */
    status: VerifyRun.status;
};

export namespace VerifyRun {

    /**
     * status of the run
     */
    export enum status {
        IN_PROGRESS = 'IN_PROGRESS',
        SUCCESS = 'SUCCESS',
        FAILURE = 'FAILURE',
    }


}

