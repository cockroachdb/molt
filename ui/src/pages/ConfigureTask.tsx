import { FormEvent, useState } from 'react';
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
import { ExpandMore } from '@material-ui/icons';
import { InputGroup, SelectCard, SelectGroup } from '../components';
import { SelectCardProps } from '../components/SelectCard';

const compressionTypes = ["None", "GZIP"] as const;
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

const booleanTexts = ["No", "Yes"] as const;

const isCloudStore = (is: IntermediateStore) => {
    return is === "GCS" || is === "S3";
}

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
    compression: "GZIP",
    flushNumRows: 0,
    flushSize: 0,
    numConcurrentTables: 4,
    numBatchRowsExport: 100_000,
    pgLogicalSlotName: "",
    pgLogicalSlotPlugin: "",
    dropPgLogicalSlot: false,
};

export default function ConfigureTask() {
    const [formState, setFormState] = useState<TaskFormState>(defaultFormState);

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
            m: 4
        }}>
            <Typography variant='h4'>Configure Task</Typography>
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
                                flexDirection: "row",
                                alignItems: "stretch",
                                gap: 2,
                            }}>
                                {
                                    modeCardDetails.map(item => {
                                        return <SelectCard
                                            key={item.id}
                                            sx={{
                                                width: "33%",
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
                                                    compression: mode !== "import" ? "None" : "GZIP"
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
                                    flexDirection: "row",
                                    alignItems: "stretch",
                                    gap: 2,
                                }}>
                                    {
                                        storesCardDetails.map(item => {
                                            return <SelectCard
                                                key={item.id}
                                                sx={{
                                                    width: "33%",
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
                                <SelectGroup
                                    required
                                    label="Cleanup intermediary store?"
                                    id="truncate"
                                    value={formState.cleanup ? booleanTexts[1] : booleanTexts[0]}
                                    onChange={(event: SelectChangeEvent) => {
                                        setFormState({
                                            ...formState,
                                            cleanup: event.target.value === booleanTexts[1]
                                        })
                                    }}
                                >
                                    {booleanTexts.map(item => {
                                        return <MenuItem key={item} value={item}>{item}</MenuItem>
                                    })}
                                </SelectGroup>
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
                                <SelectGroup
                                    required
                                    label="Truncate tables (before running fetch)"
                                    id="truncate"
                                    value={formState.truncate ? booleanTexts[1] : booleanTexts[0]}
                                    onChange={(event: SelectChangeEvent) => {
                                        setFormState({
                                            ...formState,
                                            truncate: event.target.value === booleanTexts[1]
                                        })
                                    }}
                                >
                                    {booleanTexts.map(item => {
                                        return <MenuItem key={item} value={item}>{item}</MenuItem>
                                    })}
                                </SelectGroup>
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
                                    label="Number of tables to process concurrently (size based on num CPUs)"
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
                                <SelectGroup
                                    label="Drop logical replication slot (if exists)"
                                    id="dropPgLogicalSlot"
                                    value={formState.dropPgLogicalSlot ? booleanTexts[1] : booleanTexts[0]}
                                    onChange={(event: SelectChangeEvent) => {
                                        setFormState({
                                            ...formState,
                                            dropPgLogicalSlot: event.target.value === booleanTexts[1]
                                        })
                                    }}
                                >
                                    {booleanTexts.map(item => {
                                        return <MenuItem key={item} value={item}>{item}</MenuItem>
                                    })}
                                </SelectGroup>
                            </Box>
                        </AccordionDetails>
                    </Accordion>
                    <Button sx={{ mt: 2 }} type="submit" variant="contained">Run Task</Button>
                </Box>
            </form >
        </Box >
    )
}
