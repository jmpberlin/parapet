import { describe, test, expect } from 'vitest';
import { render } from '@testing-library/react';
import DashboardCard from './DashboardCard';

describe('DashboardCard', () => {
  test('renders without props', () => {
    const { container } = render(<DashboardCard>Content</DashboardCard>);
    const card = container.querySelector('.dashboard-card');

    expect(card).toBeInTheDocument();
    expect(card).toHaveClass('dashboard-card');
    expect(card?.textContent).toContain('Content');
  });

  test('renders title when provided', () => {
    const { container } = render(
      <DashboardCard title="Test Title">Content</DashboardCard>
    );
    const header = container.querySelector('.dashboard-card__header');
    const title = container.querySelector('.dashboard-card__title');

    expect(header).toBeInTheDocument();
    expect(title?.textContent).toBe('Test Title');
  });

  test('renders count badge when provided', () => {
    const { container } = render(
      <DashboardCard title="Test Title" count={42}>Content</DashboardCard>
    );
    const count = container.querySelector('.dashboard-card__count');

    expect(count).toBeInTheDocument();
    expect(count?.textContent).toBe('42');
  });

  test('does not render header when title is undefined', () => {
    const { container } = render(<DashboardCard>Content</DashboardCard>);
    const header = container.querySelector('.dashboard-card__header');

    expect(header).not.toBeInTheDocument();
  });

  test('does not render count when count is undefined', () => {
    const { container } = render(
      <DashboardCard title="Test Title">Content</DashboardCard>
    );
    const count = container.querySelector('.dashboard-card__count');

    expect(count).not.toBeInTheDocument();
  });

  test('applies scrollable class to body when scrollable prop is true', () => {
    const { container } = render(
      <DashboardCard scrollable>Content</DashboardCard>
    );
    const body = container.querySelector('.dashboard-card__body');

    expect(body).toHaveClass('dashboard-card__body--scrollable');
  });

  test('does not apply scrollable class when scrollable prop is false or undefined', () => {
    const { container } = render(
      <DashboardCard>Content</DashboardCard>
    );
    const body = container.querySelector('.dashboard-card__body');

    expect(body).not.toHaveClass('dashboard-card__body--scrollable');
  });

  test('applies fixed-height class when fixedHeight prop is true', () => {
    const { container } = render(
      <DashboardCard fixedHeight>Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');

    expect(card).toHaveClass('dashboard-card--fixed-height');
  });

  test('does not apply fixed-height class when fixedHeight prop is false or undefined', () => {
    const { container } = render(
      <DashboardCard>Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');

    expect(card).not.toHaveClass('dashboard-card--fixed-height');
  });

  test('applies both fixedHeight and scrollable classes independently', () => {
    const { container } = render(
      <DashboardCard fixedHeight scrollable>Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');
    const body = container.querySelector('.dashboard-card__body');

    expect(card).toHaveClass('dashboard-card--fixed-height');
    expect(body).toHaveClass('dashboard-card__body--scrollable');
  });

  test('applies custom className in addition to base classes', () => {
    const { container } = render(
      <DashboardCard className="custom-class">Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');

    expect(card).toHaveClass('dashboard-card');
    expect(card).toHaveClass('custom-class');
  });

  test('applies custom className with fixedHeight and scrollable', () => {
    const { container } = render(
      <DashboardCard fixedHeight scrollable className="custom-class">Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');
    const body = container.querySelector('.dashboard-card__body');

    expect(card).toHaveClass('dashboard-card');
    expect(card).toHaveClass('dashboard-card--fixed-height');
    expect(card).toHaveClass('custom-class');
    expect(body).toHaveClass('dashboard-card__body--scrollable');
  });

  test('renders children correctly in body', () => {
    const { container } = render(
      <DashboardCard>
        <span data-testid="test-child">Test Child</span>
      </DashboardCard>
    );
    const body = container.querySelector('.dashboard-card__body') as HTMLElement | null;
    const child = container.querySelector('[data-testid="test-child"]') as HTMLElement | null;

    expect(body).toContainElement(child);
    expect(child?.textContent).toBe('Test Child');
  });

  test('maintains flex layout structure with fixedHeight', () => {
    const { container } = render(
      <DashboardCard fixedHeight title="Title">Content</DashboardCard>
    );
    const card = container.querySelector('.dashboard-card');
    const header = container.querySelector('.dashboard-card__header');
    const body = container.querySelector('.dashboard-card__body');

    expect(card).toBeInTheDocument();
    expect(header).toBeInTheDocument();
    expect(body).toBeInTheDocument();
    expect(header?.parentElement).toBe(card);
    expect(body?.parentElement).toBe(card);
  });

  test('count of 0 renders the count badge', () => {
    const { container } = render(
      <DashboardCard title="Test Title" count={0}>Content</DashboardCard>
    );
    const count = container.querySelector('.dashboard-card__count');

    expect(count).toBeInTheDocument();
    expect(count?.textContent).toBe('0');
  });

  test('supports multiple children elements', () => {
    const { container } = render(
      <DashboardCard>
        <div data-testid="child-1">Child 1</div>
        <div data-testid="child-2">Child 2</div>
      </DashboardCard>
    );
    const body = container.querySelector('.dashboard-card__body') as HTMLElement | null;
    const child1 = container.querySelector('[data-testid="child-1"]') as HTMLElement | null;
    const child2 = container.querySelector('[data-testid="child-2"]') as HTMLElement | null;

    expect(body).toContainElement(child1);
    expect(body).toContainElement(child2);
  });
});
