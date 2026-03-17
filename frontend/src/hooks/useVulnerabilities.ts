import { useQuery } from '@tanstack/react-query';
import { getVulnerabilities } from '../api';

export function useVulnerabilities() {
  return useQuery({
    queryKey: ['vulnerabilities'],
    queryFn: getVulnerabilities,
  });
}
