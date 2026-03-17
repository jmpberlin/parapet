import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getPipelineStatus, runPipeline } from '../api';

export function usePipelineStatus() {
  return useQuery({
    queryKey: ['pipeline', 'status'],
    queryFn: getPipelineStatus,
    refetchInterval: 10000, // poll every 10 seconds
  });
}

export function useRunPipeline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: runPipeline,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline', 'status'] });
    },
  });
}
