export const formatSecondsToHHMMSS = (totalSeconds: number): string => {
    let hours = Math.floor(totalSeconds / 3600);
    totalSeconds %= 3600;
    let minutes = Math.floor(totalSeconds / 60);
    let seconds = totalSeconds % 60;

    let mm = String(minutes).padStart(2, "0");
    let hh = String(hours).padStart(2, "0");
    let ss = seconds.toFixed(0).padStart(2, "0");

    return `${hh}h ${mm}m ${ss} s`;
}

export const formatNetDurationSeconds = (startedAt: number, endedAt: number, netDurationMs?: number, exportDurationMs?: number): string => {
    if (exportDurationMs !== undefined && exportDurationMs > 0) {
        return `${formatSecondsToHHMMSS(exportDurationMs / 1000)}`
    } else if (netDurationMs !== undefined && netDurationMs > 0) {
        return `${formatSecondsToHHMMSS(netDurationMs / 1000)}`
    } else if (startedAt > 0 && endedAt > 0) {
        return `${formatSecondsToHHMMSS(endedAt - startedAt)}`
    }

    return ""
}