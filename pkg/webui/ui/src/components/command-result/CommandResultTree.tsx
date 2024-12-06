import * as React from 'react';
import { useState } from 'react';
import TreeView from '@mui/lab/TreeView';
import { TreeItem } from "@mui/lab";
import { NodeData } from "./nodes/NodeData";
import { Box, Divider, useTheme } from '@mui/material';
import { TriangleDownIcon, TriangleRightIcon } from '../../icons/Icons';
import { CommandResultNodeData } from "./nodes/CommandResultNode";
import { useAppContext } from "../App";

export interface CommandResultTreeProps {
    rootNode: CommandResultNodeData

    onSelectNode: (node?: NodeData) => void
}

const CommandResultTree = (props: CommandResultTreeProps) => {
    const theme = useTheme();
    const appContext = useAppContext()
    const [expanded, setExpanded] = useState<string[]>(["root"]);

    const handleToggle = (event: React.SyntheticEvent, nodeIds: string[]) => {
        setExpanded(nodeIds);
    };

    const handleDoubleClick = (e: React.SyntheticEvent, node: NodeData) => {
        if (expanded.includes(node.id)) {
            setExpanded(expanded.filter((item) => item !== node.id));
        } else {
            setExpanded([...expanded, node.id]);
        }
        e.stopPropagation()
    };

    const handleItemClick = (e: React.SyntheticEvent, node: NodeData) => {
        props.onSelectNode(node);
        e.stopPropagation();
    }

    const renderTree = (nodes: NodeData) => {
        return <TreeItem
            key={nodes.id}
            nodeId={nodes.id}
            label={
                <Box
                    display='flex'
                    alignItems='center'
                    onClick={(e: React.SyntheticEvent) => handleItemClick(e, nodes)}
                    pl='22px'
                    height='100%'
                    flex='1 1 auto'
                    position='relative'
                >
                    {nodes.children.length !== 0 &&
                        <Divider
                            orientation='vertical'
                            sx={{
                                height: '40px',
                                position: 'absolute',
                                left: 0
                            }}
                        />
                    }
                    {nodes.buildTreeItem(appContext, nodes.children.length !== 0)}
                </Box>
            }
            sx={{
                '& .MuiTreeItem-content': {
                    height: '78px',
                    borderBottom: `0.5px solid ${theme.palette.secondary.main}`,
                    padding: 0,
                    overflow: 'hidden',
                    '& .MuiTreeItem-iconContainer': {
                        width: '50px',
                        height: '50px',
                        flex: '0 0 auto',
                        margin: 0,
                        padding: 0,
                        display: nodes.children.length !== 0 ? 'flex' : 'none',
                        justifyContent: 'center',
                        alignItems: 'center',
                    },
                    '& .MuiTreeItem-label': {
                        height: '100%',
                        margin: 0,
                        padding: 0,
                        flex: '1 1 auto',
                        display: 'flex',
                        alignItems: 'center'
                    }
                },
                '& .MuiTreeItem-group': {
                    margin: '0 0 0 38px'
                },
            }}
            onDoubleClick={(e: React.SyntheticEvent) => handleDoubleClick(e, nodes)}
        >
            {Array.isArray(nodes.children)
                ? nodes.children.map((node) => renderTree(node))
                : null}
        </TreeItem>
    };

    return <TreeView expanded={expanded}
                     onNodeToggle={handleToggle}
                     aria-label="rich object"
                     defaultCollapseIcon={<TriangleDownIcon/>}
                     defaultExpandIcon={<TriangleRightIcon/>}
                     sx={{ width: "100%" }}
    >
        {renderTree(props.rootNode)}
    </TreeView>
}

export default CommandResultTree;
