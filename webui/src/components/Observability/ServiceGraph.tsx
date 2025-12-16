/**
 * ServiceGraph Component
 *
 * Interactive service dependency graph using React Flow.
 * Displays services as nodes and dependencies as edges with health status.
 */

import React, { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  ConnectionLineType,
  MarkerType,
  Panel,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  Box,
  Paper,
  Typography,
  CircularProgress,
  Alert,
  Chip,
  Card,
  CardContent,
  IconButton,
  Tooltip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import ZoomInIcon from '@mui/icons-material/ZoomIn';
import ZoomOutIcon from '@mui/icons-material/ZoomOut';
import FitScreenIcon from '@mui/icons-material/FitScreen';
import { getServiceGraph } from '../../services/observabilityApi';
import { ServiceDependency, ServiceNode, TimeRange } from '../../services/observabilityTypes';

interface ServiceGraphProps {
  timeRange?: TimeRange;
  autoRefresh?: boolean;
  refreshInterval?: number; // seconds
}

// Custom node component
const ServiceNodeComponent = ({ data }: any) => {
  const getHealthColor = (health: string) => {
    switch (health) {
      case 'healthy':
        return '#4caf50';
      case 'degraded':
        return '#ff9800';
      case 'unhealthy':
        return '#f44336';
      default:
        return '#9e9e9e';
    }
  };

  return (
    <Card
      sx={{
        minWidth: 200,
        border: `2px solid ${getHealthColor(data.health)}`,
        borderRadius: 2,
        boxShadow: 3,
      }}
    >
      <CardContent sx={{ p: 2 }}>
        <Typography variant="h6" gutterBottom>
          {data.name}
        </Typography>
        <Chip
          label={data.health}
          size="small"
          sx={{
            bgcolor: getHealthColor(data.health),
            color: 'white',
            mb: 1,
          }}
        />
        <Typography variant="body2" color="text.secondary">
          {data.requestRate.toFixed(2)} req/s
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Error: {data.errorRate.toFixed(2)}%
        </Typography>
        <Typography variant="body2" color="text.secondary">
          p95: {data.p95Latency.toFixed(0)}ms
        </Typography>
      </CardContent>
    </Card>
  );
};

const nodeTypes = {
  serviceNode: ServiceNodeComponent,
};

export const ServiceGraph: React.FC<ServiceGraphProps> = ({
  timeRange,
  autoRefresh = false,
  refreshInterval = 30,
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);

  const loadServiceGraph = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const graphData = await getServiceGraph(timeRange);
      const graph: ServiceDependency = Array.isArray(graphData) ? graphData[0] : graphData;

      // Convert service nodes to React Flow nodes
      const flowNodes: Node[] = graph.nodes.map((node: ServiceNode, index) => ({
        id: node.id,
        type: 'serviceNode',
        position: { x: 0, y: 0 }, // Will be auto-laid out
        data: {
          name: node.name,
          health: node.health,
          requestRate: node.requestRate,
          errorRate: node.errorRate,
          p95Latency: node.p95Latency,
          type: node.type,
        },
      }));

      // Convert service edges to React Flow edges
      const flowEdges: Edge[] = graph.edges.map((edge, index) => ({
        id: `${edge.source}-${edge.target}`,
        source: edge.source,
        target: edge.target,
        type: ConnectionLineType.SmoothStep,
        animated: edge.requestRate > 10,
        label: `${edge.requestRate.toFixed(1)} req/s`,
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
        style: {
          strokeWidth: Math.min(edge.requestRate / 10 + 1, 5),
          stroke: edge.errorRate > 5 ? '#f44336' : '#90caf9',
        },
      }));

      // Apply dagre layout
      const layoutedNodes = applyDagreLayout(flowNodes, flowEdges);
      setNodes(layoutedNodes);
      setEdges(flowEdges);
    } catch (err: any) {
      setError(err.message || 'Failed to load service graph');
      console.error('Error loading service graph:', err);
    } finally {
      setLoading(false);
    }
  }, [timeRange, setNodes, setEdges]);

  // Auto-layout using dagre
  const applyDagreLayout = (nodes: Node[], edges: Edge[]): Node[] => {
    // Simple grid layout as fallback
    const gridSize = Math.ceil(Math.sqrt(nodes.length));
    return nodes.map((node, index) => ({
      ...node,
      position: {
        x: (index % gridSize) * 300,
        y: Math.floor(index / gridSize) * 200,
      },
    }));
  };

  // Initial load
  useEffect(() => {
    loadServiceGraph();
  }, [loadServiceGraph]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh && refreshInterval > 0) {
      const interval = setInterval(() => {
        loadServiceGraph();
      }, refreshInterval * 1000);
      return () => clearInterval(interval);
    }
  }, [autoRefresh, refreshInterval, loadServiceGraph]);

  const handleFitView = () => {
    if (reactFlowInstance) {
      reactFlowInstance.fitView({ padding: 0.2 });
    }
  };

  if (loading && nodes.length === 0) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Paper sx={{ height: 600, position: 'relative' }}>
      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onInit={setReactFlowInstance}
        nodeTypes={nodeTypes}
        connectionLineType={ConnectionLineType.SmoothStep}
        fitView
      >
        <Background />
        <Controls />
        <Panel position="top-right">
          <Box display="flex" gap={1}>
            <Tooltip title="Refresh">
              <IconButton
                size="small"
                onClick={loadServiceGraph}
                disabled={loading}
                sx={{ bgcolor: 'background.paper' }}
              >
                <RefreshIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title="Fit to view">
              <IconButton
                size="small"
                onClick={handleFitView}
                sx={{ bgcolor: 'background.paper' }}
              >
                <FitScreenIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Panel>
      </ReactFlow>
    </Paper>
  );
};

export default ServiceGraph;
