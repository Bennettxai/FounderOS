import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	getAppTemplates,
	getAppTemplate,
	getBuiltInTemplates,
	getTemplateRecommendations,
	generateAppFromTemplate
} from './templates';

// Mock the request function from base
vi.mock('../base', () => ({
	API_BASE: 'http://localhost:8080/api',
	request: vi.fn()
}));

import { request } from '../base';

describe('Templates API', () => {
	const mockTemplate = {
		id: 'template-1',
		name: 'CRM System',
		description: 'Customer relationship management',
		category: 'crm',
		business_type: 'saas',
		features: ['contacts', 'deals', 'pipeline'],
		estimated_time_hours: 40,
		complexity: 'medium' as const,
		created_at: '2024-01-01T00:00:00Z'
	};

	const mockTemplates = [
		mockTemplate,
		{
			id: 'template-2',
			name: 'Dashboard',
			description: 'Analytics dashboard',
			category: 'analytics',
			business_type: 'saas',
			features: ['charts', 'metrics'],
			estimated_time_hours: 20,
			complexity: 'low' as const,
			created_at: '2024-01-02T00:00:00Z'
		}
	];

	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe('getAppTemplates()', () => {
		it('should fetch all templates without filters', async () => {
			(request as any).mockResolvedValueOnce({
				templates: mockTemplates,
				total: 2
			});

			const result = await getAppTemplates();

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates',
				{ method: 'GET' }
			);
			expect(result.templates).toEqual(mockTemplates);
			expect(result.total).toBe(2);
		});

		it('should fetch templates with category filter', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [mockTemplate],
				total: 1
			});

			await getAppTemplates({ category: 'crm' });

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates?category=crm',
				{ method: 'GET' }
			);
		});

		it('should fetch templates with business_type filter', async () => {
			(request as any).mockResolvedValueOnce({
				templates: mockTemplates,
				total: 2
			});

			await getAppTemplates({ business_type: 'saas' });

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates?business_type=saas',
				{ method: 'GET' }
			);
		});

		it('should fetch templates with search query', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [mockTemplate],
				total: 1
			});

			await getAppTemplates({ search: 'crm' });

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates?search=crm',
				{ method: 'GET' }
			);
		});

		it('should fetch templates with pagination', async () => {
			(request as any).mockResolvedValueOnce({
				templates: mockTemplates,
				total: 100
			});

			await getAppTemplates({ limit: 10, offset: 20 });

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates?limit=10&offset=20',
				{ method: 'GET' }
			);
		});

		it('should fetch templates with multiple filters', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [mockTemplate],
				total: 1
			});

			await getAppTemplates({
				category: 'crm',
				business_type: 'saas',
				search: 'crm',
				sort: 'popular',
				limit: 5
			});

			expect(request).toHaveBeenCalledWith(
				expect.stringContaining('category=crm'),
				{ method: 'GET' }
			);
			expect(request).toHaveBeenCalledWith(
				expect.stringContaining('business_type=saas'),
				{ method: 'GET' }
			);
			expect(request).toHaveBeenCalledWith(
				expect.stringContaining('search=crm'),
				{ method: 'GET' }
			);
		});

		it('should handle empty results', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [],
				total: 0
			});

			const result = await getAppTemplates();

			expect(result.templates).toEqual([]);
			expect(result.total).toBe(0);
		});

		it('should handle API error', async () => {
			(request as any).mockRejectedValueOnce(new Error('Network error'));

			await expect(getAppTemplates()).rejects.toThrow('Network error');
		});
	});

	describe('getAppTemplate()', () => {
		it('should fetch template by ID', async () => {
			(request as any).mockResolvedValueOnce(mockTemplate);

			const result = await getAppTemplate('template-1');

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates/template-1',
				{ method: 'GET' }
			);
			expect(result).toEqual(mockTemplate);
		});

		it('should handle template not found', async () => {
			(request as any).mockRejectedValueOnce(new Error('Template not found'));

			await expect(getAppTemplate('non-existent')).rejects.toThrow('Template not found');
		});

		it('should handle invalid ID format', async () => {
			(request as any).mockRejectedValueOnce(new Error('Invalid template ID'));

			await expect(getAppTemplate('')).rejects.toThrow();
		});
	});

	describe('getBuiltInTemplates()', () => {
		const mockBuiltInTemplates = {
			templates: [
				{
					id: 'crm',
					name: 'CRM System',
					description: 'Built-in CRM',
					category: 'crm',
					config_schema: { type: 'object' }
				},
				{
					id: 'dashboard',
					name: 'Dashboard',
					description: 'Built-in dashboard',
					category: 'project_management',
					config_schema: { type: 'object' }
				}
			]
		};

		it('should fetch built-in templates', async () => {
			(request as any).mockResolvedValueOnce(mockBuiltInTemplates);

			const result = await getBuiltInTemplates();

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates/builtin',
				{ method: 'GET' }
			);
			expect(result.templates).toHaveLength(2);
			expect(result.templates[0].id).toBe('crm');
		});

		it('should handle empty built-in templates', async () => {
			(request as any).mockResolvedValueOnce({ templates: [] });

			const result = await getBuiltInTemplates();

			expect(result.templates).toEqual([]);
		});

		it('should handle API error', async () => {
			(request as any).mockRejectedValueOnce(new Error('Server error'));

			await expect(getBuiltInTemplates()).rejects.toThrow('Server error');
		});
	});

	describe('getTemplateRecommendations()', () => {
		const mockRecommendations = [
			{
				template_id: 'crm',
				template_name: 'CRM System',
				match_score: 0.95,
				reasoning: 'Perfect for your business type'
			},
			{
				template_id: 'dashboard',
				template_name: 'Dashboard',
				match_score: 0.80,
				reasoning: 'Good analytics fit'
			}
		];

		it('should fetch recommendations for workspace', async () => {
			(request as any).mockResolvedValueOnce(mockRecommendations);

			const result = await getTemplateRecommendations('workspace-123');

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/workspaces/workspace-123/template-recommendations',
				{ method: 'GET' }
			);
			expect(result).toHaveLength(2);
			expect(result[0].match_score).toBe(0.95);
		});

		it('should handle no recommendations', async () => {
			(request as any).mockResolvedValueOnce([]);

			const result = await getTemplateRecommendations('workspace-123');

			expect(result).toEqual([]);
		});

		it('should handle workspace not found', async () => {
			(request as any).mockRejectedValueOnce(new Error('Workspace not found'));

			await expect(getTemplateRecommendations('invalid')).rejects.toThrow('Workspace not found');
		});
	});

	describe('generateAppFromTemplate()', () => {
		const mockGenerationRequest = {
			workspace_id: 'workspace-123',
			app_name: 'My CRM',
			config: {
				enable_contacts: true,
				enable_deals: true,
				primary_color: '#3B82F6',
				max_users: 100
			}
		};

		const mockGenerationResult = {
			message: 'App generated successfully',
			result: {
				app_id: 'app-456',
				status: 'completed' as const,
				files_created: ['src/app.ts', 'src/components/Contact.tsx'],
				estimated_completion_time: 300
			}
		};

		it('should generate app from template', async () => {
			(request as any).mockResolvedValueOnce(mockGenerationResult);

			const result = await generateAppFromTemplate('crm', mockGenerationRequest);

			expect(request).toHaveBeenCalledWith(
				'http://localhost:8080/api/app-templates/crm/generate',
				{
					method: 'POST',
					body: JSON.stringify(mockGenerationRequest)
				}
			);
			expect(result.result!.app_id).toBe('app-456');
			expect(result.result!.status).toBe('completed');
		});

		it('should handle generation with minimal config', async () => {
			const minimalRequest = {
				workspace_id: 'workspace-123',
				app_name: 'Simple App',
				config: {}
			};

			(request as any).mockResolvedValueOnce({
				message: 'App generated',
				result: {
					app_id: 'app-789',
					status: 'completed' as const,
					files_created: [],
					estimated_completion_time: 60
				}
			});

			const result = await generateAppFromTemplate('dashboard', minimalRequest);

			expect(result.result!.app_id).toBe('app-789');
		});

		it('should handle validation errors', async () => {
			(request as any).mockRejectedValueOnce(
				new Error('Validation error: workspace_id is required')
			);

			await expect(
				generateAppFromTemplate('crm', {} as any)
			).rejects.toThrow('Validation error');
		});

		it('should handle template not found', async () => {
			(request as any).mockRejectedValueOnce(new Error('Template not found'));

			await expect(
				generateAppFromTemplate('non-existent', mockGenerationRequest)
			).rejects.toThrow('Template not found');
		});

		it('should handle generation failure', async () => {
			(request as any).mockRejectedValueOnce(
				new Error('Generation failed: insufficient permissions')
			);

			await expect(
				generateAppFromTemplate('crm', mockGenerationRequest)
			).rejects.toThrow('Generation failed');
		});
	});

	describe('Edge cases', () => {
		it('should handle very long search queries', async () => {
			const longQuery = 'a'.repeat(500);
			(request as any).mockResolvedValueOnce({
				templates: [],
				total: 0
			});

			await getAppTemplates({ search: longQuery });

			expect(request).toHaveBeenCalledWith(
				expect.stringContaining('search='),
				{ method: 'GET' }
			);
		});

		it('should handle special characters in search', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [],
				total: 0
			});

			await getAppTemplates({ search: 'test & <script>alert("xss")</script>' });

			expect(request).toHaveBeenCalled();
		});

		it('should handle large offset values', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [],
				total: 0
			});

			await getAppTemplates({ offset: 1000000 });

			expect(request).toHaveBeenCalledWith(
				expect.stringContaining('offset=1000000'),
				{ method: 'GET' }
			);
		});

		it('should handle concurrent requests', async () => {
			(request as any).mockResolvedValue({
				templates: mockTemplates,
				total: 2
			});

			const promises = [
				getAppTemplates(),
				getAppTemplates({ category: 'crm' }),
				getAppTemplates({ limit: 5 })
			];

			await Promise.all(promises);

			expect(request).toHaveBeenCalledTimes(3);
		});

		it('should handle empty search filter', async () => {
			(request as any).mockResolvedValueOnce({
				templates: mockTemplates,
				total: 2
			});

			await getAppTemplates({ search: '' });

			// Empty strings should still be included in URL
			expect(request).toHaveBeenCalled();
		});
	});

	describe('Response format validation', () => {
		it('should handle response with extra fields', async () => {
			(request as any).mockResolvedValueOnce({
				templates: mockTemplates,
				total: 2,
				extra_field: 'extra',
				metadata: { foo: 'bar' }
			});

			const result = await getAppTemplates();

			expect(result.templates).toEqual(mockTemplates);
			expect((result as any).extra_field).toBe('extra');
		});

		it('should handle malformed template data', async () => {
			(request as any).mockResolvedValueOnce({
				templates: [{ id: 'test' }], // Missing required fields
				total: 1
			});

			const result = await getAppTemplates();

			expect(result.templates).toHaveLength(1);
		});
	});
});
