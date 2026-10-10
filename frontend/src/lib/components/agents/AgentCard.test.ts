import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import AgentCard from './AgentCard.svelte';
import type { CustomAgent } from '$lib/api/ai/types';

describe('AgentCard Component', () => {
  const mockAgent: CustomAgent = {
    id: '1',
    user_id: 'user1',
    name: 'test-agent',
    display_name: 'Test Agent',
    description: 'A test agent for testing',
    system_prompt: 'You are a test agent',
    category: 'general',
    model_preference: 'gpt-4',
    is_active: true,
    times_used: 42,
    created_at: '2024-01-01',
    updated_at: '2024-01-01'
  };

  const mockAgentWithAvatar: CustomAgent = {
    ...mockAgent,
    id: '2',
    avatar: 'https://example.com/avatar.png'
  };

  const inactiveAgent: CustomAgent = {
    ...mockAgent,
    id: '3',
    is_active: false
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render agent card with all basic information', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      expect(screen.getByText('Test Agent')).toBeTruthy();
      expect(screen.getByText('@test-agent')).toBeTruthy();
      expect(screen.getByText('A test agent for testing')).toBeTruthy();
      expect(screen.getByText('General')).toBeTruthy();
      expect(screen.getByText('gpt-4')).toBeTruthy();
      expect(screen.getByText('42 uses')).toBeTruthy();
      expect(container).toBeTruthy();
    });

    it('should render active status badge', () => {
      render(AgentCard, {
        props: { agent: mockAgent }
      });

      const activeIndicator = screen.getByText('Active');
      expect(activeIndicator).toBeTruthy();
    });

    it('should render inactive status badge', () => {
      render(AgentCard, {
        props: { agent: inactiveAgent }
      });

      const inactiveIndicator = screen.getByText('Inactive');
      expect(inactiveIndicator).toBeTruthy();
    });

    it('should render avatar image when provided', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgentWithAvatar }
      });

      const avatarImg = container.querySelector('img[alt="Test Agent"]');
      expect(avatarImg).toBeTruthy();
      expect(avatarImg?.getAttribute('src')).toBe('https://example.com/avatar.png');
    });

    it('should render initials when no avatar provided', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      // Should render "TA" for "Test Agent"
      expect(screen.getByText('TA')).toBeTruthy();
    });

    it('should handle agent without description', () => {
      const agentNoDesc = { ...mockAgent, description: undefined };
      render(AgentCard, {
        props: { agent: agentNoDesc }
      });

      expect(screen.getByText('No description provided.')).toBeTruthy();
    });

    it('should not render model badge if no model preference', () => {
      const agentNoModel = { ...mockAgent, model_preference: undefined };
      const { container } = render(AgentCard, {
        props: { agent: agentNoModel }
      });

      expect(container.textContent).not.toContain('gpt-4');
    });

    it('should not render usage count badge if zero or undefined', () => {
      const agentNoUsage = { ...mockAgent, times_used: 0 };
      const { container } = render(AgentCard, {
        props: { agent: agentNoUsage }
      });

      // Should not find the usage count badge
      expect(container.querySelector('svg[viewBox="0 0 24 24"]')?.parentElement?.textContent).not.toBe(
        '0'
      );
    });

    it('should render in compact variant', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent, variant: 'compact' }
      });

      const card = container.querySelector('.compact');
      expect(card).toBeTruthy();
    });
  });

  describe('Interactions', () => {
    it('should call onSelect when Select button is clicked', async () => {
      const onSelect = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onSelect }
      });

      await fireEvent.click(screen.getByRole('button', { name: 'Select Test Agent' }));
      expect(onSelect).toHaveBeenCalledWith(mockAgent);
    });

    it('should render Select button when onSelect is provided', () => {
      render(AgentCard, {
        props: { agent: mockAgent, onSelect: vi.fn() }
      });

      const selectButton = screen.getByText('Select');
      expect(selectButton).toBeTruthy();
    });

    it('should not render Select button when onSelect is not provided', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      expect(container.textContent).not.toContain('Select');
    });

    it('should open menu when menu button is clicked', async () => {
      render(AgentCard, {
        props: { agent: mockAgent, onEdit: vi.fn(), onDelete: vi.fn() }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      expect(screen.getByText('Edit')).toBeTruthy();
      expect(screen.getByText('Delete')).toBeTruthy();
    });

    it('should call onEdit when Edit button is clicked', async () => {
      const onEdit = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onEdit, onDelete: vi.fn() }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      const editButton = screen.getByText('Edit');
      await fireEvent.click(editButton);

      expect(onEdit).toHaveBeenCalledWith(mockAgent);
    });

    it('should show delete confirmation on first delete click', async () => {
      const onDelete = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onEdit: vi.fn(), onDelete }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      const deleteButton = screen.getByText('Delete');
      await fireEvent.click(deleteButton);

      expect(screen.getByText('Are you sure? This cannot be undone.')).toBeTruthy();
      expect(onDelete).not.toHaveBeenCalled();
    });

    it('should call onDelete when confirmed', async () => {
      const onDelete = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onEdit: vi.fn(), onDelete }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      let deleteButton = screen.getByText('Delete');
      await fireEvent.click(deleteButton);

      // Now click the confirmation Delete button
      const buttons = screen.getAllByText('Delete');
      const confirmButton = buttons[buttons.length - 1]; // Get the last "Delete" button (confirmation)
      await fireEvent.click(confirmButton);

      expect(onDelete).toHaveBeenCalledWith(mockAgent);
    });

    it('should cancel delete confirmation', async () => {
      const onDelete = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onEdit: vi.fn(), onDelete }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      const deleteButton = screen.getByText('Delete');
      await fireEvent.click(deleteButton);

      const cancelButton = screen.getByText('Cancel');
      await fireEvent.click(cancelButton);

      expect(onDelete).not.toHaveBeenCalled();
      // Menu should be closed
      expect(screen.queryByText('Are you sure?')).toBeFalsy();
    });

    it('should not show menu button when no edit/delete handlers', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      expect(container.querySelector('[aria-label="More actions"]')).toBeFalsy();
    });

    it('should prevent event propagation when clicking menu', async () => {
      const onSelect = vi.fn();
      render(AgentCard, {
        props: { agent: mockAgent, onSelect, onEdit: vi.fn() }
      });

      const menuButton = screen.getByLabelText('More actions');
      await fireEvent.click(menuButton);

      // onSelect should not be called when clicking the menu
      expect(onSelect).not.toHaveBeenCalled();
    });

    it('should prevent event propagation when clicking Select button', async () => {
      const onSelect = vi.fn();
      const { container } = render(AgentCard, {
        props: { agent: mockAgent, onSelect }
      });

      const selectButton = screen.getByText('Select');
      await fireEvent.click(selectButton);

      // Should call onSelect exactly once (not twice from card click)
      expect(onSelect).toHaveBeenCalledTimes(1);
    });
  });

  describe('Helper Functions', () => {
    it('should generate correct initials for single word', () => {
      const singleWordAgent = { ...mockAgent, display_name: 'Agent' };
      render(AgentCard, {
        props: { agent: singleWordAgent }
      });

      expect(screen.getByText('AG')).toBeTruthy();
    });

    it('should generate correct initials for multi-word names', () => {
      const multiWordAgent = { ...mockAgent, display_name: 'Super Awesome Agent' };
      render(AgentCard, {
        props: { agent: multiWordAgent }
      });

      expect(screen.getByText('SA')).toBeTruthy(); // Only first 2 initials
    });

    it('should apply correct category color classes', () => {
      const { container, rerender } = render(AgentCard, {
        props: { agent: { ...mockAgent, category: 'general' } }
      });

      expect(container.textContent).toContain('General');

      // Test different categories
      rerender({ agent: { ...mockAgent, category: 'specialist' } });
      expect(container.textContent).toContain('Specialist');

      rerender({ agent: { ...mockAgent, category: 'custom' } });
      expect(container.textContent).toContain('Custom');

      rerender({ agent: { ...mockAgent, category: undefined } });
      // Should not have any category badge
    });
  });

  describe('Accessibility', () => {
    it('should have an accessible Select button when selectable', () => {
      render(AgentCard, {
        props: { agent: mockAgent, onSelect: vi.fn() }
      });

      expect(screen.getByRole('button', { name: 'Select Test Agent' })).toBeTruthy();
    });

    it('should not be keyboard accessible when not selectable', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      expect(container.querySelector('[role="button"]')).toBeFalsy();
    });

    it('should have proper aria-label for menu button', () => {
      render(AgentCard, {
        props: { agent: mockAgent, onEdit: vi.fn() }
      });

      const menuButton = screen.getByLabelText('More actions');
      expect(menuButton).toBeTruthy();
    });

    it('should have alt text for avatar images', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgentWithAvatar }
      });

      const img = container.querySelector('img');
      expect(img?.getAttribute('alt')).toBe('Test Agent');
    });

    it('should have title attribute for description tooltip', () => {
      const { container } = render(AgentCard, {
        props: { agent: mockAgent }
      });

      const description = container.querySelector('.line-clamp-2');
      expect(description?.getAttribute('title')).toBe('A test agent for testing');
    });
  });

  describe('Edge Cases', () => {
    it('should handle very long agent names', () => {
      const longNameAgent = {
        ...mockAgent,
        display_name: 'This is a very long agent name that should be truncated'
      };

      const { container } = render(AgentCard, {
        props: { agent: longNameAgent }
      });

      expect(container.textContent).toContain('This is a very long agent name');
    });

    it('should handle very long descriptions', () => {
      const longDescAgent = {
        ...mockAgent,
        description: 'This is a very long description '.repeat(20)
      };

      const { container } = render(AgentCard, {
        props: { agent: longDescAgent }
      });

      // Description should be clamped by CSS
      const description = container.querySelector('.line-clamp-2');
      expect(description).toBeTruthy();
    });

    it('should handle agent with no category', () => {
      const noCategoryAgent = { ...mockAgent, category: undefined };
      const { container } = render(AgentCard, {
        props: { agent: noCategoryAgent }
      });

      // Should still render without errors
      expect(screen.getByText('Test Agent')).toBeTruthy();
    });

    it('should handle agent with empty string name', () => {
      const emptyNameAgent = { ...mockAgent, display_name: '', name: 'empty' };
      const { container } = render(AgentCard, {
        props: { agent: emptyNameAgent }
      });

      expect(container.textContent).toContain('@empty');
    });

    it('should handle all callback props being undefined', () => {
      const { container } = render(AgentCard, {
        props: {
          agent: mockAgent,
          onSelect: undefined,
          onEdit: undefined,
          onDelete: undefined
        }
      });

      // Should render without errors
      expect(container).toBeTruthy();
    });
  });
});
