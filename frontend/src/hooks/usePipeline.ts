import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getPipelineStatus, runPipeline } from '../api';

export function usePipelineStatus() {
  return useQuery({
    queryKey: ['pipeline', 'status'],
    queryFn: getPipelineStatus,
    refetchInterval: (query) =>
      query.state.data?.run_in_progress ? 1000 : 10000,
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
