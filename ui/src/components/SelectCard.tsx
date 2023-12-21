import {
    Box,
    Card,
    CardProps,
    CardActionArea,
    CardActions,
    CardContent,
    Button,
    Typography
} from "@mui/material";
import { green } from '@mui/material/colors';
import { CheckCircle } from '@material-ui/icons';


export type SelectCardProps = {
    id: string;
    title: string;
    description: string;
    link: string;
    isSelected?: boolean;
    onClick?: React.MouseEventHandler<HTMLButtonElement>;
}
type CombinedSelectCardProps = SelectCardProps & CardProps;

export default function SelectCard(props: CombinedSelectCardProps) {
    const { id, title, description, link, onClick, isSelected, ...rest } = props;
    return (
        <Card id={id} {...rest}>
            <CardActionArea onClick={onClick}>
                <CardContent>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        alignItems: "center",
                        gap: 0.5,
                    }}>
                        <Typography sx={{ width: "90%" }} variant="h6" component="div">
                            {title}
                        </Typography>
                        {isSelected && <CheckCircle style={{ color: green[500] }} />}
                    </Box>
                    <Typography variant="body2">
                        {description}
                    </Typography>
                </CardContent>
                <CardActions>
                    <Button href={link} target="_blank" size="small">Learn More</Button>
                </CardActions>
            </CardActionArea>
        </Card >);
}
