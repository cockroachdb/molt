import { CreateFetchTaskRequestBody, MoltserviceService, OpenAPIConfig } from "../apigen";

const BASE_URL = "http://localhost:4500"

// TODO: configure the OpenAPI config.

export const getFetchTasks = () => {
    return MoltserviceService.moltserviceGetFetchTasks()
}

export const getSpecificFetchTask = (id: number) => {
    return MoltserviceService.moltserviceGetSpecificFetchTask(id)
}

// TODO: update this later
export const createFetchTask = () => {
    //const reqBody = CreateFetchTaskRequestBody
    //return MoltserviceService.moltserviceCreateFetchTask();
}
