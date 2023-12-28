import { FormEvent, useEffect, useState } from 'react';
import { useNavigate } from "react-router-dom";
import {
    Typography,
    Box,
    MenuItem,
    Button,
    SelectChangeEvent,
    Accordion,
    AccordionSummary,
    AccordionDetails
} from '@mui/material';
import { grey } from '@mui/material/colors';
import { MuiMarkdown, getOverrides } from 'mui-markdown';
import { ExpandMore } from '@material-ui/icons';
import CodeMirror from "@uiw/react-codemirror";

import { InputGroup, SelectCard, SelectGroup, Switch } from '../components';
import { SelectCardProps } from '../components/cards/SelectCard';
import { neutral } from '../styles/colors';
import { HOME_PATH } from '.';

const compressionTypes = ["default", "none", "gzip"] as const;
type CompressionType = typeof compressionTypes[number];

const modeTypes = ["import", "directCopy", "liveCopyFromStore"] as const;
type Mode = typeof modeTypes[number];
const modeCardDetails: SelectCardProps[] = [
    {
        id: "import",
        title: "IMPORT into (intermediate store)",
        description: "Recommended path because the load is more efficient and supports compression. The downside is that the target table is taken offline.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
    {
        id: "directCopy",
        title: "Direct copy from source",
        description: "Leaves the target table online while the data load is ongoing. Allows for a data movement between source and target without using an intermediary store like AWS S3.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
    {
        id: "liveCopyFromStore",
        title: "Live copy from intermediate store",
        description: "COPY FROM an intermediate store to the target.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
]

const intermediateStores = ["local", "S3", "GCS"] as const;
type IntermediateStore = typeof intermediateStores[number];
const storesCardDetails: SelectCardProps[] = [
    {
        id: "local",
        title: "Local store",
        description: "Runs a local file server and uses it as the intermediate store.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
    {
        id: "S3",
        title: "Amazon S3",
        description: "Use an existing AWS S3 bucket as the intermediate store.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
    {
        id: "GCS",
        title: "Google Cloud GCS",
        description: "Use an existing GCP GCS bucket as the intermediate store.",
        link: "https://github.com/cockroachdb/molt/blob/main/docs/RFCS/20231113_molt_fetch.md#options",
    },
]

const isCloudStore = (is: IntermediateStore) => {
    return is === "GCS" || is === "S3";
}

const configureTaskMD = `
##### Mode of Operation
The mode of operation dictates at a high level how the data export/import will run and which mechanisms it will use. These will balance 
tradeoffs of performance vs. disk/RAM usage vs. if target tables are taken offline or not.

<br/>

##### Intermediate Store
The intermediate store applies to the case where an \`IMPORT INTO\` or \`COPY FROM\` CSV/Avro files is run to move the data to the target.
It is recommended to use a cloud storage platform here so that you can size down the disk of your machine running \`fetch\`.

For each case, you must provide relevant details.

General:
- **Cleanup intermediate store**: if set, cleans up the intermediate store by removing all files created during the fetch task. Leave on if you want to debug the data later.

Local:
- **Local path**: the absolute (\`/usr/Documents/fetch\`) or relative (\`./fetch\`) path to write export files.
- **Local path listen address**: the local address where the file server will be spun up
- **Local path CRDB access address**: the local address CRDB will use to access the import files

Cloud:
- **Bucket name**: the name of the bucket in the cloud storage provider (make sure this is accessible via your cloud credentials)
- **Bucket path**: the sub-path within the bucket to write the export files (i.e. \`fetch/export\`)

<br/>
##### Task Level Settings
Task level settings relate to overall task execution and actions that apply to the holistic run.

- **Log File**: if specified, writes task execution logs to a file and stdout; otherwise, just writes to stdout
- **Compression Type**: None is the default for \`direct-copy\` and \`live\` modes; however for \`IMPORT INTO\` GZIP is the default to allow for much quicker imports
- **Truncate Tables**: if specified, truncates the target tables before running the data load; allows for a clean slate before data movement, preventing data collisions

<br/>
##### Performance Tuning
Performance tuning settings allow the user to specify batch sizes for export/tables/etc., parallelism parameters, and 
flush parameters.

- **Number of rows before flushing data**: number of rows for the export before data is flushed to the disk (persisted)
- **Number of bytes before flushing data**: number of bytes for the export before data is flushed to the disk
- **Number of tables to process concurrently**: number of tables to process at the same time; this is usually sized based on number of CPUs; 4 is the default
- **Number of rows to export**: number of rows to export at a given time for each iteration; tune this so that you get most out of CPU and can batch the most data together

<br/>
##### Replication Settings
Replication settings allow the user to specify slot names, plugins, and relevant behavior if existing slots exist. Currently, only applies to PostgreSQL.

- **Replication Slot Name**: name for the replication slot
- **Replication Slot Plugin**: name for the replication plugin
- **Drop logical replication slot**: if set and exists, drops the existing replication slot
`

interface TaskFormState {
    mode: Mode,
    store: IntermediateStore,
    bucketName: string,
    bucketPath: string,
    localPath: string,
    localPathListenAddr: string,
    localPathCRDBAccessAddr: string,
    logFile: string,
    cleanup: boolean,
    truncate: boolean,
    compression: CompressionType,
    flushNumRows: number,
    flushSize: number,
    numConcurrentTables: number,
    numBatchRowsExport: number,
    pgLogicalSlotName: string,
    pgLogicalSlotPlugin: string,
    dropPgLogicalSlot: boolean,
}

const defaultFormState: TaskFormState = {
    mode: "import",
    store: "local",
    bucketName: "",
    bucketPath: "",
    localPath: "",
    localPathListenAddr: "",
    localPathCRDBAccessAddr: "",
    logFile: "",
    cleanup: false,
    truncate: false,
    compression: "gzip",
    flushNumRows: 0,
    flushSize: 0,
    numConcurrentTables: 4,
    numBatchRowsExport: 100_000,
    pgLogicalSlotName: "",
    pgLogicalSlotPlugin: "",
    dropPgLogicalSlot: false,
};

const mockSource = "postgres://postgres@localhost:5432/postgres"
const mockTarget = "postgres://root@localhost:26257/defaultdb?sslmode=disable"
const moltFetchCmd = "molt fetch"
const getFetchCmdFromTaskFormState = (tf: TaskFormState, source: string, target: string) => {
    let cmd = moltFetchCmd;

    cmd = `${cmd}\n --source ${source} \\\n`
    cmd = `${cmd} --target ${target}`

    // mode
    if (tf.mode === "directCopy") {
        cmd = `${cmd} --direct-copy`
    } else if (tf.mode === "liveCopyFromStore") {
        cmd = `${cmd} --live`
    }
    cmd = `${cmd} \\\n`

    // intermediate store
    if (tf.mode !== 'directCopy') {
        if (tf.store === "local") {
            cmd = `${cmd} --local-path ${tf.localPath} \\\n --local-path-listen-addr ${tf.localPathListenAddr} \\\n --local-path-crdb-access-addr ${tf.localPathCRDBAccessAddr}`
        } else if (isCloudStore(tf.store)) {
            let bucketNameFlag = "--gcp-bucket"
            if (tf.store === "S3") {
                bucketNameFlag = "--s3-bucket"
            }

            cmd = `${cmd} ${bucketNameFlag} ${tf.bucketName}`

            if (tf.bucketPath.trim() !== "") {
                cmd = `${cmd} --bucket-path ${tf.bucketPath}`
            }
        }

        if (tf.cleanup) {
            cmd = `${cmd} --cleanup`
        }
        cmd = `${cmd} \\\n`
    }

    // task level setting
    cmd = `${cmd} --compression ${tf.compression}`
    if (tf.logFile.trim() !== "") {
        cmd = `${cmd} --log-file ${tf.logFile}`
    }
    if (tf.truncate) {
        cmd = `${cmd} --truncate`
    }
    cmd = `${cmd} \\\n`

    // performance tuning
    if (tf.flushNumRows > 0) {
        cmd = `${cmd} --flush-rows ${tf.flushNumRows}`
    }
    if (tf.flushSize > 0) {
        cmd = `${cmd} --flush-size ${tf.flushSize}`
    }
    if (tf.numConcurrentTables > 0) {
        cmd = `${cmd} --concurrency ${tf.numConcurrentTables}`
    }
    if (tf.numBatchRowsExport > 0) {
        cmd = `${cmd} --row-batch-size ${tf.numBatchRowsExport}`
    }
    cmd = `${cmd} \\\n`

    // TODO: make it so that this only triggers on postgresql
    // replication settings
    if (tf.pgLogicalSlotName.trim() !== "") {
        cmd = `${cmd} --pg-logical-replication-slot-name ${tf.pgLogicalSlotName} \\\n`
    }
    if (tf.pgLogicalSlotPlugin.trim() !== "") {
        cmd = `${cmd} --pg-logical-replication-slot-plugin ${tf.pgLogicalSlotPlugin} \\\n`
    }
    if (tf.dropPgLogicalSlot) {
        cmd = `${cmd} --pg-logical-replication-slot-drop-if-exists`
    }

    return cmd.trimEnd().endsWith(`\\`) ? cmd.trimEnd().slice(0, -1) : cmd.trimEnd();
}

const cardMediaQuery = '@media screen and (min-width: 1200px)';

export default function ConfigureTask() {
    const navigate = useNavigate();
    const [formState, setFormState] = useState<TaskFormState>(defaultFormState);
    const [outputCmd, setOutputCmd] = useState<string>(getFetchCmdFromTaskFormState(defaultFormState, mockSource, mockTarget));

    useEffect(() => {
        setOutputCmd(getFetchCmdFromTaskFormState(formState, mockSource, mockTarget));
    }, [formState])

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        setFormState({
            ...formState,
            [e.target.id]: e.target.value,
        });
    }

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        console.log(formState)
    }

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "row",
            justifyContent: "center",
            height: "100%"
        }}>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                alignItems: "stretch",
                flex: 1,
                py: 4,
                px: 10,
                maxWidth: "50%"
            }}>
                <Typography sx={{ mb: 1 }} variant='h4'>Configure Task</Typography>
                <form onSubmit={handleSubmit}>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "column",
                    }}>
                        <Accordion defaultExpanded>
                            <AccordionSummary
                                expandIcon={<ExpandMore />}
                                aria-controls="mode-panel"
                                id="mode-panel"
                            >
                                <Typography>Mode of Operation</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Box sx={{
                                    display: "flex",
                                    flexDirection: "column",
                                    alignItems: "center",
                                    justifyContent: "stretch",
                                    gap: 2,
                                    [cardMediaQuery]: {
                                        flexDirection: "row",
                                        alignItems: "stretch"
                                    },
                                }}>
                                    {
                                        modeCardDetails.map(item => {
                                            return <SelectCard
                                                key={item.id}
                                                sx={{
                                                    width: "90%",
                                                    [cardMediaQuery]: {
                                                        width: "33%",
                                                    },
                                                }}
                                                id={item.id}
                                                title={item.title}
                                                description={item.description}
                                                link={item.link}
                                                isSelected={formState.mode === item.id}
                                                onClick={(e) => {
                                                    const mode = item.id as Mode;
                                                    setFormState({
                                                        ...formState,
                                                        mode: mode,
                                                        compression: mode !== "import" ? "none" : "gzip"
                                                    });
                                                }}
                                            />
                                        })
                                    }
                                </Box>
                            </AccordionDetails>
                        </Accordion>
                        {formState.mode !== "directCopy" && <Accordion defaultExpanded>
                            <AccordionSummary
                                expandIcon={<ExpandMore />}
                                aria-controls="intermediate-store-panel"
                                id="intermediate-store-panel"
                            >
                                <Typography>Intermediate Store</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Box sx={{
                                    display: "flex",
                                    flexDirection: "column",
                                    gap: 3
                                }}>
                                    <Box sx={{
                                        display: "flex",
                                        flexDirection: "column",
                                        alignItems: "center",
                                        justifyContent: "stretch",
                                        gap: 2,
                                        [cardMediaQuery]: {
                                            flexDirection: "row",
                                            alignItems: "stretch"
                                        },
                                    }}>
                                        {
                                            storesCardDetails.map(item => {
                                                return <SelectCard
                                                    key={item.id}
                                                    sx={{
                                                        width: "90%",
                                                        [cardMediaQuery]: {
                                                            width: "33%",
                                                        },
                                                    }}
                                                    id={item.id}
                                                    title={item.title}
                                                    description={item.description}
                                                    link={item.link}
                                                    isSelected={formState.store === item.id}
                                                    onClick={(e) => {
                                                        setFormState({
                                                            ...formState,
                                                            store: item.id as IntermediateStore
                                                        });
                                                    }}
                                                />
                                            })
                                        }
                                    </Box>
                                    {isCloudStore(formState.store) && <Box
                                        id="cloudStore"
                                        sx={{
                                            display: "flex",
                                            flexDirection: "column",
                                            gap: 3
                                        }}>
                                        <InputGroup
                                            label="Bucket name"
                                            id="bucketName"
                                            value={formState.bucketName}
                                            validation={(value) => {
                                                if (value.length === 0) return "Field cannot be empty."

                                                return ""
                                            }}
                                            onChange={handleInputChange} />
                                        <InputGroup
                                            label="Bucket path"
                                            id="bucketPath"
                                            value={formState.bucketPath}
                                            validation={(value) => {
                                                if (value.length === 0) return "Field cannot be empty."

                                                return ""
                                            }}
                                            onChange={handleInputChange} />
                                    </Box>}
                                    {!isCloudStore(formState.store) && <Box
                                        id="localStore"
                                        sx={{
                                            display: "flex",
                                            flexDirection: "column",
                                            gap: 3
                                        }}>
                                        <InputGroup
                                            label="Local path"
                                            id="localPath"
                                            value={formState.localPath}
                                            validation={(value) => {
                                                if (value.length === 0) return "Field cannot be empty."

                                                return ""
                                            }}
                                            onChange={handleInputChange} />
                                        <InputGroup
                                            label="Local path listen address"
                                            id="localPathListenAddr"
                                            value={formState.localPathListenAddr}
                                            validation={(value) => {
                                                if (value.length === 0) return "Field cannot be empty."

                                                return ""
                                            }}
                                            onChange={handleInputChange} />
                                        <InputGroup
                                            label="Local path CRDB access address"
                                            id="localPathCRDBAccessAddr"
                                            value={formState.localPathCRDBAccessAddr}
                                            validation={(value) => {
                                                if (value.length === 0) return "Field cannot be empty."

                                                return ""
                                            }}
                                            onChange={handleInputChange} />
                                    </Box>}
                                    <Switch
                                        required
                                        label="Cleanup intermediary store?"
                                        id="truncate"
                                        value={formState.cleanup}
                                        onChange={(event: SelectChangeEvent) => {
                                            setFormState({
                                                ...formState,
                                                cleanup: !formState.cleanup
                                            })
                                        }}
                                    />
                                </Box>
                            </AccordionDetails>
                        </Accordion>}
                        <Accordion defaultExpanded>
                            <AccordionSummary
                                expandIcon={<ExpandMore />}
                                aria-controls="task-panel"
                                id="task-panel"
                            >
                                <Typography>Task Level Settings</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Box sx={{
                                    display: "flex",
                                    flexDirection: "column",
                                    gap: 3
                                }}>
                                    <InputGroup
                                        label="Log file"
                                        id="logFile"
                                        value={formState.logFile}
                                        validation={(value) => { return "" }}
                                        onChange={handleInputChange} />
                                    <SelectGroup
                                        required
                                        label="Compression type"
                                        id="compression"
                                        value={formState.compression}
                                        onChange={(event: SelectChangeEvent) => {
                                            setFormState({
                                                ...formState,
                                                compression: event.target.value as CompressionType
                                            })
                                        }}
                                    >
                                        {compressionTypes.map(item => {
                                            return <MenuItem key={item} value={item}>{item}</MenuItem>
                                        })}
                                    </SelectGroup>
                                    <Switch
                                        required
                                        label="Truncate tables (before running fetch)"
                                        id="truncate"
                                        value={formState.truncate}
                                        onChange={(_: SelectChangeEvent) => {
                                            setFormState({
                                                ...formState,
                                                truncate: !formState.truncate
                                            })
                                        }}
                                    />
                                </Box>
                            </AccordionDetails>
                        </Accordion>
                        <Accordion defaultExpanded>
                            <AccordionSummary
                                expandIcon={<ExpandMore />}
                                aria-controls="performance-panel"
                                id="performance-panel"
                            >
                                <Typography>Performance Tuning</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Box sx={{
                                    display: "flex",
                                    flexDirection: "column",
                                    gap: 3
                                }}>
                                    <InputGroup
                                        label="Number of rows before flushing data"
                                        id="flushNumRows"
                                        value={formState.flushNumRows}
                                        type="number"
                                        validation={(value) => {
                                            if (value.length === 0) return "Field cannot be empty."

                                            return ""
                                        }}
                                        onChange={handleInputChange} />
                                    <InputGroup
                                        label="Number of bytes before flushing data"
                                        id="flushSize"
                                        value={formState.flushSize}
                                        type="number"
                                        validation={(value) => {
                                            if (value.length === 0) return "Field cannot be empty."

                                            return ""
                                        }}
                                        onChange={handleInputChange} />
                                    <InputGroup
                                        label="Number of tables to process concurrently"
                                        id="numConcurrentTables"
                                        value={formState.numConcurrentTables}
                                        type="number"
                                        validation={(value) => {
                                            if (value.length === 0) return "Field cannot be empty."

                                            return ""
                                        }}
                                        onChange={handleInputChange} />
                                    <InputGroup
                                        label="Number of rows to export at a time from the source"
                                        id="numBatchRowsExport"
                                        value={formState.numBatchRowsExport}
                                        type="number"
                                        validation={(value) => {
                                            if (value.length === 0) return "Field cannot be empty."

                                            return ""
                                        }}
                                        onChange={handleInputChange} />
                                </Box>
                            </AccordionDetails>
                        </Accordion>
                        <Accordion defaultExpanded>
                            <AccordionSummary
                                expandIcon={<ExpandMore />}
                                aria-controls="replication-panel"
                                id="replication-panel"
                            >
                                <Typography>Replication Settings</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Box sx={{
                                    display: "flex",
                                    flexDirection: "column",
                                    gap: 3
                                }}>
                                    <InputGroup
                                        label="PG logical replication slot name"
                                        id="pgLogicalSlotName"
                                        value={formState.pgLogicalSlotName}
                                        validation={(value) => { return "" }}
                                        onChange={handleInputChange} />
                                    <InputGroup
                                        label="PG logical replication slot plugin"
                                        id="pgLogicalSlotPlugin"
                                        value={formState.pgLogicalSlotPlugin}
                                        validation={(value) => { return "" }}
                                        onChange={handleInputChange} />
                                    <Switch
                                        label="Drop logical replication slot (if exists)"
                                        id="dropPgLogicalSlot"
                                        value={formState.dropPgLogicalSlot}
                                        onChange={(_: SelectChangeEvent) => {
                                            setFormState({
                                                ...formState,
                                                dropPgLogicalSlot: !formState.dropPgLogicalSlot
                                            })
                                        }}
                                    />
                                </Box>
                            </AccordionDetails>
                        </Accordion>
                        <Box sx={{ my: 2 }} >
                            <CodeMirror
                                value={outputCmd}
                                height="100px"
                            />
                        </Box>
                        <Button onClick={() => navigate(HOME_PATH)} type="submit" variant="contained">Run Task</Button>
                    </Box>
                </form >
            </Box>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                flex: 1,
                backgroundColor: neutral[100],
                py: 4,
                px: 6,
                gap: 4,
                maxWidth: "50%"
            }}>
                <Typography variant="h4">Setup Guide</Typography>
                <MuiMarkdown overrides={{
                    ...getOverrides(), // This will keep the other default overrides.
                    code: {
                        props: {
                            style: { fontSize: "0.8rem", backgroundColor: grey[200] },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                    li: {
                        props: {
                            style: { fontSize: "0.9rem" },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                    p: {
                        props: {
                            style: { fontSize: "0.9rem" },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                    strong: {
                        props: {
                            style: { fontSize: "0.95rem" },
                        } as React.HTMLProps<HTMLParagraphElement>,
                    },
                }}>
                    {configureTaskMD}
                </MuiMarkdown>
            </Box>
        </Box >
    )
}
