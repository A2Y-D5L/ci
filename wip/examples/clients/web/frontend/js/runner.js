// CI Pipeline Dashboard - Plan Runner Class

import { escapeHtml } from './utils.js';
import * as api from './api.js';

/**
 * PlanRunner handles loading, displaying, and running saved plans.
 * Supports running all targets, specific targets, or specific stages.
 */
export class PlanRunner {
    constructor() {
        this.plans = [];
        this.selectedPlan = null;
        this.runMode = 'all'; // 'all', 'target', 'stage'
        this.selectedTarget = null;
        this.selectedStage = null;
        this.failTargets = new Set();
        
        this.init();
    }

    async init() {
        await this.loadPlans();
    }

    async loadPlans() {
        try {
            const data = await api.fetchSavedPlans();
            this.plans = data.plans || [];
            this.renderPlans();
        } catch (err) {
            console.error('Failed to load plans:', err);
            this.showError('Failed to load plans');
        }
    }

    refreshPlans() {
        this.loadPlans();
    }

    renderPlans() {
        const container = document.getElementById('savedPlansList');
        if (!container) return;

        if (this.plans.length === 0) {
            container.innerHTML = `
                <div class="empty-plans">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <path d="M9 12h6M9 16h6M17 21H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                    </svg>
                    <p>No saved plans yet.<br>Create one in the Composer tab!</p>
                </div>
            `;
            return;
        }

        container.innerHTML = this.plans.map(plan => `
            <div class="plan-card ${plan.id === this.selectedPlan?.id ? 'selected' : ''} ${plan.isBuiltin ? 'builtin' : 'user-created'}" 
                 onclick="window.planRunner.selectPlan('${plan.id}')">
                ${!plan.isBuiltin ? `<button class="plan-card-delete" onclick="event.stopPropagation(); window.planRunner.deletePlan('${plan.id}')" title="Delete plan">✕</button>` : ''}
                <div class="plan-card-header">
                    <span class="plan-card-name">${escapeHtml(plan.name)}</span>
                    <span class="plan-card-badge ${plan.isBuiltin ? 'builtin' : 'user'}">${plan.isBuiltin ? 'Built-in' : 'Custom'}</span>
                </div>
                <div class="plan-card-description">${escapeHtml(plan.description || 'No description')}</div>
                <div class="plan-card-meta">
                    <span>${plan.targetCount || plan.targets?.length || 0} targets</span>
                    <span>${plan.stageCount || 0} stages</span>
                </div>
            </div>
        `).join('');
    }

    async selectPlan(planId) {
        try {
            const data = await api.fetchPlanDetails(planId);
            
            if (data.status === 'error') {
                this.showError(data.message || 'Failed to load plan details');
                return;
            }
            
            if (data.status === 'success' || data.status === 'ok') {
                this.selectedPlan = {
                    ...data.plan,
                    stages: data.stages?.map(s => s.targets) || [],
                    targets: data.targets || []
                };
                this.runMode = 'all';
                this.selectedTarget = null;
                this.selectedStage = null;
                this.failTargets.clear();
                
                this.renderPlans();
                this.showPlanDetails();
            }
        } catch (err) {
            console.error('Failed to load plan details:', err);
            this.showError('Failed to load plan details: ' + err.message);
        }
    }

    showPlanDetails() {
        const panel = document.getElementById('planDetailsPanel');
        if (!panel || !this.selectedPlan) return;

        panel.style.display = 'block';
        
        document.getElementById('selectedPlanName').textContent = this.selectedPlan.name;
        document.getElementById('selectedPlanDescription').textContent = 
            this.selectedPlan.description || 'No description provided.';

        // Reset mode buttons
        document.querySelectorAll('.mode-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === this.runMode);
        });

        // Render fail targets selector
        this.renderFailTargets();
        
        // Render stages preview
        this.renderStagesPreview();

        // Hide target/stage selector initially
        document.getElementById('targetStageSelector').style.display = 'none';
    }

    setRunMode(mode) {
        this.runMode = mode;
        this.selectedTarget = null;
        this.selectedStage = null;

        // Update button states
        document.querySelectorAll('.mode-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === mode);
        });

        const selector = document.getElementById('targetStageSelector');
        const selectorLabel = document.getElementById('selectorLabel');
        const selectorOptions = document.getElementById('selectorOptions');

        if (mode === 'all') {
            selector.style.display = 'none';
            this.renderStagesPreview();
        } else if (mode === 'target') {
            selector.style.display = 'block';
            selectorLabel.textContent = 'Select Target';
            this.renderTargetSelector(selectorOptions);
        } else if (mode === 'stage') {
            selector.style.display = 'block';
            selectorLabel.textContent = 'Select Stage';
            this.renderStageSelector(selectorOptions);
        }
    }

    renderTargetSelector(container) {
        if (!this.selectedPlan?.stages) return;

        const allTargets = new Set();
        this.selectedPlan.stages.forEach(stage => {
            stage.forEach(target => allTargets.add(target));
        });

        container.innerHTML = Array.from(allTargets).map(target => `
            <div class="selector-option ${this.selectedTarget === target ? 'selected' : ''}" 
                 onclick="window.planRunner.selectTarget('${target}')">
                ${target}
            </div>
        `).join('');
    }

    renderStageSelector(container) {
        if (!this.selectedPlan?.stages) return;

        container.innerHTML = this.selectedPlan.stages.map((stage, idx) => `
            <div class="selector-option ${this.selectedStage === idx ? 'selected' : ''}" 
                 onclick="window.planRunner.selectStage(${idx})">
                Stage ${idx + 1}: ${stage.join(', ')}
            </div>
        `).join('');
    }

    selectTarget(target) {
        this.selectedTarget = target;
        this.renderTargetSelector(document.getElementById('selectorOptions'));
        this.renderStagesPreview();
    }

    selectStage(stageIdx) {
        this.selectedStage = stageIdx;
        this.renderStageSelector(document.getElementById('selectorOptions'));
        this.renderStagesPreview();
    }

    renderFailTargets() {
        const container = document.getElementById('planFailSelector');
        if (!container || !this.selectedPlan?.stages) return;

        const allTargets = new Set();
        this.selectedPlan.stages.forEach(stage => {
            stage.forEach(target => allTargets.add(target));
        });

        if (allTargets.size === 0) {
            container.innerHTML = '<div class="empty-hint">No targets available</div>';
            return;
        }

        container.innerHTML = Array.from(allTargets).map(target => `
            <div class="fail-option ${this.failTargets.has(target) ? 'selected' : ''}"
                 onclick="window.planRunner.toggleFailTarget('${target}')">
                ${target}
            </div>
        `).join('');
    }

    toggleFailTarget(target) {
        if (this.failTargets.has(target)) {
            this.failTargets.delete(target);
        } else {
            this.failTargets.add(target);
        }
        this.renderFailTargets();
    }

    renderStagesPreview() {
        const container = document.getElementById('planStagesPreview');
        if (!container || !this.selectedPlan?.stages) return;

        container.innerHTML = this.selectedPlan.stages.map((stage, idx) => {
            const isHighlighted = this.runMode === 'stage' && this.selectedStage === idx;
            const hasSelectedTarget = this.runMode === 'target' && stage.includes(this.selectedTarget);
            
            return `
                <div class="preview-stage ${isHighlighted || hasSelectedTarget ? 'highlighted' : ''}">
                    <span class="preview-stage-number">${idx + 1}</span>
                    <div class="preview-stage-targets">
                        ${stage.map(target => {
                            const isTargetSelected = this.runMode === 'target' && target === this.selectedTarget;
                            return `<span class="preview-target ${isTargetSelected ? 'highlighted' : ''}">${target}</span>`;
                        }).join('')}
                    </div>
                </div>
            `;
        }).join('');
    }

    closePlanDetails() {
        const panel = document.getElementById('planDetailsPanel');
        if (panel) {
            panel.style.display = 'none';
        }
        this.selectedPlan = null;
        this.renderPlans();
    }

    async deletePlan(planId) {
        if (!confirm('Are you sure you want to delete this plan?')) return;

        try {
            const data = await api.deletePlan(planId);
            
            if (data.status === 'deleted') {
                if (this.selectedPlan?.id === planId) {
                    this.closePlanDetails();
                }
                this.loadPlans();
            } else {
                alert(data.message || 'Failed to delete plan');
            }
        } catch (err) {
            console.error('Failed to delete plan:', err);
            alert('Failed to delete plan');
        }
    }

    async runSelectedPlan(dashboard) {
        if (!this.selectedPlan) return;

        const request = {
            planId: this.selectedPlan.id,
            mode: this.runMode,
            failTargets: Array.from(this.failTargets)
        };

        if (this.runMode === 'target' && this.selectedTarget) {
            request.targetName = this.selectedTarget;
        } else if (this.runMode === 'stage' && this.selectedStage !== null) {
            request.stageIndex = this.selectedStage;
        }

        try {
            // Clear previous run
            if (dashboard) {
                dashboard.clear();
            }

            const data = await api.startPlanRun(request);
            
            if (data.status !== 'started') {
                alert(data.message || 'Failed to start plan');
            }
        } catch (err) {
            console.error('Failed to run plan:', err);
            alert('Failed to run plan: ' + err.message);
        }
    }

    showError(message) {
        const container = document.getElementById('savedPlansList');
        if (!container) return;

        container.innerHTML = `
            <div class="error-message">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M12 2v20m10-10H2"/>
                </svg>
                <div class="error-text">${message}</div>
            </div>
        `;
    }
}
