import { describe, test, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import MatchDetail from './MatchDetail';
import * as useRepositoriesHook from '../../hooks/useRepositories';
import type { components } from '../../types/api';

vi.mock('../../hooks/useRepositories');

type MatchDetailData = components['schemas']['MatchDetail'];
type UseMatchDetailResult = ReturnType<typeof useRepositoriesHook.useMatchDetail>;

const mockUseMatchDetail = (overrides: Partial<UseMatchDetailResult> = {}) => {
  vi.mocked(useRepositoriesHook.useMatchDetail).mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
    ...overrides,
  } as UseMatchDetailResult);
};

const detail: MatchDetailData = {
  match: {
    id: 'match-1',
    vulnerability_id: 'vuln-1',
    repository_id: 'repo-1',
    component_purl: 'pkg:npm/lodash@4.17.15',
    matched_component: 'lodash',
    matched_version: '4.17.15',
    status: 'CONFIRMED',
    created_at: '2024-01-02T00:00:00Z',
    matched_on: 'normalized_name',
    vuln_identifier: 'lodash',
    dep_identifier: 'lodash',
    confidence: 'CONFIRMED',
    version_check: 'in_range',
  },
  dependency: {
    name: 'lodash',
    version: '4.17.15',
    purl: 'pkg:npm/lodash@4.17.15',
  },
  vulnerability: {
    id: 'vuln-1',
    cve: 'CVE-2020-8203',
    severity: 'HIGH',
    description: 'Prototype pollution in lodash before 4.17.21.',
    affected_technologies: [
      { name: 'lodash', purl: 'pkg:npm/lodash', version_range: '<4.17.21' },
    ],
    published_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
  },
  source_articles: [
    {
      id: 'article-1',
      headline: 'Lodash prototype pollution disclosed',
      source_url: 'https://www.bleepingcomputer.com/news/security/lodash',
      host_domain: 'bleepingcomputer.com',
      published_at: '2024-01-01T00:00:00Z',
    },
  ],
};

const renderPage = (id: string) =>
  render(
    <MemoryRouter initialEntries={[id]}>
      <Routes>
        <Route path='/matches/:id' element={<MatchDetail />} />
      </Routes>
    </MemoryRouter>,
  );

describe('MatchDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('requests the match id from the route and renders all four sections', () => {
    mockUseMatchDetail({ data: detail });
    renderPage('/matches/match-1');

    expect(useRepositoriesHook.useMatchDetail).toHaveBeenCalledWith('match-1');

    // vulnerability section
    expect(screen.getByText('CVE-2020-8203')).toBeInTheDocument();
    expect(screen.getByText('Prototype pollution in lodash before 4.17.21.')).toBeInTheDocument();
    expect(screen.getByText('<4.17.21')).toBeInTheDocument();

    // dependency section
    expect(screen.getByText('pkg:npm/lodash@4.17.15')).toBeInTheDocument();
    expect(screen.getAllByText('lodash').length).toBeGreaterThanOrEqual(3);

    // source articles section
    const sourceLink = screen.getByRole('link', {
      name: /Lodash prototype pollution disclosed/,
    });
    expect(sourceLink).toHaveAttribute('href', '/articles/article-1');
    expect(screen.getByText(/bleepingcomputer\.com/)).toBeInTheDocument();

    // reasoning section
    expect(screen.getByText(/Matched by package name/)).toBeInTheDocument();
  });

  test('shows a plain-language reasoning line built from the evidence fields', () => {
    mockUseMatchDetail({ data: detail });
    renderPage('/matches/match-1');

    const reasoning = screen.getByText(/Matched by package name/);
    expect(reasoning.textContent).toBe(
      'Matched by package name: lodash ~ lodash. Version 4.17.15 is inside <4.17.21 → confirmed',
    );
    expect(screen.getByText('package name')).toBeInTheDocument();
    expect(screen.getByText('in range')).toBeInTheDocument();
    expect(screen.getByText('confirmed')).toBeInTheDocument();
  });

  test('shows a fallback reasoning line when the version check is unknown', () => {
    mockUseMatchDetail({
      data: {
        ...detail,
        match: { ...detail.match, version_check: undefined },
        vulnerability: {
          ...detail.vulnerability,
          affected_technologies: [
            { name: 'lodash', purl: 'pkg:npm/lodash', version_range: '' },
          ],
        },
      },
    });
    renderPage('/matches/match-1');

    expect(
      screen.getByText(/Version check could not be run → needs review/),
    ).toBeInTheDocument();
    expect(screen.getByText('unknown')).toBeInTheDocument();
  });

  test('shows a keyword tier label for word_token matches', () => {
    mockUseMatchDetail({
      data: {
        ...detail,
        match: { ...detail.match, matched_on: 'word_token', confidence: 'LOW' },
        vulnerability: {
          ...detail.vulnerability,
          affected_technologies: [
            { name: 'lodash', purl: 'pkg:npm/lodash', version_range: '' },
          ],
        },
      },
    });
    renderPage('/matches/match-1');

    expect(screen.getByText(/Matched by keyword/)).toBeInTheDocument();
    expect(screen.getByText('keyword')).toBeInTheDocument();
  });

  test('shows a loading state while fetching', () => {
    mockUseMatchDetail({ isLoading: true });
    renderPage('/matches/match-1');

    expect(screen.getByText('Loading…')).toBeInTheDocument();
  });

  test('shows an error state when the request fails', () => {
    mockUseMatchDetail({ isError: true, error: new Error('failed to fetch match') });
    renderPage('/matches/match-1');

    expect(screen.getByText(/Error: failed to fetch match/)).toBeInTheDocument();
  });

  test('shows a not-found state when there is no data', () => {
    mockUseMatchDetail({});
    renderPage('/matches/match-1');

    expect(screen.getByText('Match not found.')).toBeInTheDocument();
  });
});
