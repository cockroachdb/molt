/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type FetchRun = {
    /**
     * finished at time
     */
    finished_at: number;
    /**
     * ID of the run
     */
    id: number;
    /**
     * name of the fetch run
     */
    name: string;
    /**
     * started at time
     */
    started_at: number;
    /**
     * status of the fetch run
     */
    status: FetchRun.status;
};

export namespace FetchRun {

    /**
     * status of the fetch run
     */
    export enum status {
        IN_PROGRESS = 'IN_PROGRESS',
        SUCCESS = 'SUCCESS',
        FAILURE = 'FAILURE',
    }


}

