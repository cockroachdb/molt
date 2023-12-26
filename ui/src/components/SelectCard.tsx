import {
    Box,
    Card,
    CardProps,
    CardActionArea,
    Button,
    CardContent,
    Typography
} from "@mui/material";
import { CheckCircle } from '@material-ui/icons';
import { fontSizes, fontWeights } from "../styles/fonts";
import { DEFAULT_SPACING } from "../styles/theme";
import { info, neutral } from "../styles/colors";


export type SelectCardProps = {
    id: string;
    title: string;
    description: string;
    link: string;
    disabled?: boolean;
    isSelected?: boolean;
    onClick?: React.MouseEventHandler<HTMLButtonElement>;
}
type CombinedSelectCardProps = SelectCardProps & CardProps;

export default function SelectCard(props: CombinedSelectCardProps) {
    const { sx, disabled, id, title, description, link, onClick, isSelected, ...rest } = props;

    // Use a box shadow because it doesn't cause shift in content when
    // the width thickens or thins.
    const getBoxShadowColor = (isSelected?: boolean, isDisabled?: boolean) => {
        if (isSelected && isDisabled) {
            return `0 0 0 2px ${info[2]}`
        }

        if (isSelected && !isDisabled) {
            return `0 0 0 2px ${info[3]}`
        }

        return `0 0 0 1px ${neutral[500]}`
    }

    return (
        <Card sx={{
            border: "none",
            boxSizing: "border-box",
            boxShadow: getBoxShadowColor(isSelected, disabled),
            zIndex: 5,
            "&:hover": {
                opacity: 1,
                boxShadow: disabled ? undefined : `0 0 0 2px ${info[3]}`
            },
            ...sx
        }} id={id} {...rest} variant="outlined">
            <CardActionArea disabled={disabled} sx={{
                opacity: 1,
                "&:hover": {
                    opacity: 1
                }
            }} onClick={onClick}>
                <Box sx={{
                    display: "flex",
                    flexDirection: "row",
                    justifyContent: "flex-end",
                    height: DEFAULT_SPACING * 2,
                    pt: 1,
                    pr: 1,
                }}>
                    {isSelected && <CheckCircle style={{ color: disabled ? info[2] : info[3] }} />}
                </Box>
                <CardContent sx={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "flex-start",
                    gap: 1,
                }}>
                    <Typography sx={{ width: "90%", fontWeight: fontWeights["heavy"], fontSize: fontSizes["xl"] }} variant="h3" component="div">
                        {title}
                    </Typography>
                    <Typography sx={{ fontWeight: fontWeights["medium"], fontSize: fontSizes["sm"] }} variant="body2">
                        {description}
                    </Typography>
                    <Button sx={{
                        color: info[3],
                        padding: 0,
                        textTransform: "none"
                    }} type="link" target="_blank" href={link}>Learn More</Button>
                </CardContent>
            </CardActionArea>
        </Card >);
}
