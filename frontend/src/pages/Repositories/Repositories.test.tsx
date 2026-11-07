import { describe, test, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Repositories from './Repositories';
import * as useRepositoriesHook from '../../hooks/useRepositories';
import * as useArticlesHook from '../../hooks/useArticles';
import * as usePipelineHook from '../../hooks/usePipeline';
import type { components } from '../../types/api';

vi.mock('../../hooks/useRepositories');
vi.mock('../../hooks/useArticles');
vi.mock('../../hooks/usePipeline');

const repo1 = {
  id: 'repo-1',
  owner_name: 'jmp',
  repository_name: 'nightwatch',
  git_provider: 'Github.com',
  dependency_count: 2,
  match_count: 3,
};
const repo2 = { ...repo1, id: 'repo-2', repository_name: 'other' };

type Paginated = components['schemas']['PaginatedMatches'];

const paginated = (page: number): Paginated => ({
  items: [
    {
      id: `item-${page}`,
      vulnerability_id: 'vuln-1',
      repository_id: 'repo-1',
      component_purl: 'pkg:npm/lodash@4.17.0',
      matched_component: 'lodash',
      matched_version: '4.17.0',
      status: 'CONFIRMED',
      created_at: '2024-01-01T00:00:00Z',
    },
  ],
  total: 20,
  page,
  limit: 10,
  total_pages: 2,
});

const paginatedArticles = (page: number) => ({
  items: [
    {
      id: `article-${page}`,
      source_url: 'https://example.com/a',
      host_domain: 'bleepingcomputer.com' as const,
      published_at: '2024-01-01T00:00:00Z',
      headline: 'Security article',
      author: '',
      content_html: '',
      content_cleaned: '',
      crawled_at: '2024-01-01T00:00:00Z',
    },
  ],
  total: 20,
  page,
  limit: 10,
  total_pages: 2,
});

const setupHooks = () => {
  vi.mocked(useRepositoriesHook.useRepositories).mockReturnValue({
    data: [repo1, repo2],
    isLoading: false,
  } as never);
  vi.mocked(useRepositoriesHook.useRepository).mockReturnValue({
    data: repo1,
    isLoading: false,
  } as never);
  vi.mocked(usePipelineHook.usePipelineStatus).mockReturnValue({
    data: {
      run_in_progress: false,
      ran_at: '2024-01-01T00:00:00Z',
      articles_harvested: 0,
      vulnerabilities_extracted: 0,
      deps_added: 0,
      deps_removed: 0,
      matches_found: 0,
      errors: [],
    },
  } as never);
  vi.mocked(usePipelineHook.useRunPipeline).mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
  } as never);
};

function renderPage() {
  return render(
    <MemoryRouter>
      <Repositories />
    </MemoryRouter>,
  );
}

describe('Repositories page pagination', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupHooks();

    vi.mocked(useRepositoriesHook.useRepositoryMatches).mockReturnValue({
      data: paginated(1),
      isLoading: false,
      isFetching: false,
    } as never);
    vi.mocked(useRepositoriesHook.useRepositoryDependencies).mockReturnValue({
      data: paginated(1),
      isLoading: false,
      isFetching: false,
    } as never);
    vi.mocked(useArticlesHook.useArticlesPaginated).mockReturnValue({
      data: paginatedArticles(1),
      isLoading: false,
      isFetching: false,
    } as never);
  });

  test('renders pagination inside the cards on first load', () => {
    renderPage();

    expect(useRepositoriesHook.useRepositoryMatches).toHaveBeenCalledWith('repo-1', 1);
    expect(useArticlesHook.useArticlesPaginated).toHaveBeenCalledWith(7, 1);
    expect(screen.getAllByText('Page 1 of 2')).toHaveLength(3);
  });

  test('resets matches and dependencies pages when the repository changes', async () => {
    vi.mocked(useRepositoriesHook.useRepositoryMatches).mockImplementation(
      (_id: string, page: number) =>
        ({
          data: paginated(page),
          isLoading: false,
          isFetching: false,
        }) as never,
    );

    renderPage();

    await waitFor(() =>
      expect(useRepositoriesHook.useRepositoryMatches).toHaveBeenLastCalledWith('repo-1', 1),
    );

    fireEvent.click(screen.getAllByRole('button', { name: 'Next →' })[0]);

    await waitFor(() =>
      expect(useRepositoriesHook.useRepositoryMatches).toHaveBeenLastCalledWith('repo-1', 2),
    );

    fireEvent.click(screen.getByText('other'));

    await waitFor(() =>
      expect(useRepositoriesHook.useRepositoryMatches).toHaveBeenLastCalledWith('repo-2', 1),
    );
    await waitFor(() =>
      expect(useRepositoriesHook.useRepositoryDependencies).toHaveBeenLastCalledWith('repo-2', 1),
    );
    expect(useArticlesHook.useArticlesPaginated).toHaveBeenLastCalledWith(7, 1);
  });
});
