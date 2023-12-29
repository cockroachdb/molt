import { ReactNode } from "react";
import { MuiMarkdown, getOverrides } from 'mui-markdown';
import { neutral } from "../../styles/colors";


export type MarkdownProps = {
    children: ReactNode
};

export default function Markdown(props: MarkdownProps) {
    const { children } = props;

    return <MuiMarkdown overrides={{
        ...getOverrides(), // This will keep the other default overrides.
        code: {
            props: {
                style: { fontSize: "0.8rem", backgroundColor: neutral[200] },
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
        {children as string}
    </MuiMarkdown>
}
