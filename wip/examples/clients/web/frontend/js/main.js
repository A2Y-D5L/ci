// CI Pipeline Dashboard - Main Application Entry Point

import { CIDashboard } from './dashboard.js';
import { PlanComposer } from './composer.js';
import { PlanRunner } from './runner.js';
import * as api from './api.js';

// Global instances - exposed on window for onclick handlers in HTML
let dashboard;
let composer;
let planRunner;

/**
 * Switch between tabs (composer and execution).
 * @param {string} tabName - The tab to switch to ('composer' or 'execution').
 */
function switchTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tabName);
    });
    
    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === tabName + 'Tab');
    });
}

/**
 * Start a demo pipeline run.
 * @param {boolean} withFailure - Whether to simulate a failure.
 */
async function startRun(withFailure) {
    try {
        switchTab('execution');
        dashboard.clear();
        const data = await api.startDemoRun(withFailure);
        
        if (data.status === 'error') {
            alert(data.message);
        }
    } catch (err) {
        console.error('Failed to start run:', err);
        alert('Failed to start run: ' + err.message);
    }
}

/**
 * Cancel the current pipeline run.
 */
async function cancelRun() {
    try {
        const data = await api.cancelRun();
        
        if (data.status === 'error') {
            alert(data.message);
        }
    } catch (err) {
        console.error('Failed to cancel run:', err);
        alert('Failed to cancel run: ' + err.message);
    }
}

/**
 * Clear the dashboard.
 */
function clearDashboard() {
    if (dashboard) {
        dashboard.clear();
    }
}

/**
 * Select all targets in the composer.
 */
function selectAllTargets() {
    if (composer) {
        composer.selectAllTargets();
    }
}

/**
 * Clear target selection in the composer.
 */
function clearSelection() {
    if (composer) {
        composer.clearSelection();
    }
}

/**
 * Run the custom plan from the composer.
 */
function runCustomPlan() {
    if (composer) {
        composer.runPlan(switchTab, dashboard);
    }
}

/**
 * Save the current plan from the composer.
 */
function savePlan() {
    if (!composer || composer.selectedTargets.size === 0) {
        alert('Please select at least one target to save.');
        return;
    }

    const modal = document.getElementById('savePlanModal');
    const previewContainer = document.getElementById('savePlanPreview');
    
    const planData = composer.buildPlan();
    if (!planData || !planData.valid) {
        previewContainer.innerHTML = '<p class="error">Plan is not valid. Cannot save.</p>';
    } else {
        previewContainer.innerHTML = `
            <p><strong>${planData.targets.length} targets</strong> selected.</p>
            <p><strong>${planData.stages.length} stages</strong> will be created.</p>
            <p><strong>${planData.rootTargets.length > 0 ? planData.rootTargets.length : 'Default'} root targets</strong>.</p>
        `;
    }

    document.getElementById('planName').value = '';
    document.getElementById('planDescription').value = '';
    modal.style.display = 'flex';
}

/**
 * Close the save plan modal.
 */
function closeSavePlanModal() {
    const modal = document.getElementById('savePlanModal');
    modal.style.display = 'none';
}

/**
 * Confirm and save the plan from the modal.
 */
async function confirmSavePlan() {
    const name = document.getElementById('planName').value.trim();
    if (!name) {
        alert('Plan name is required.');
        return;
    }

    const description = document.getElementById('planDescription').value.trim();
    const planData = composer.buildPlan();

    if (!planData || !planData.valid) {
        alert('Cannot save an invalid or empty plan.');
        return;
    }

    try {
        const result = await api.savePlan({
            name: name,
            description: description,
            targets: planData.targets,
        });

        if (result.status === 'saved') {
            alert(`Plan "${result.plan.name}" saved successfully!`);
            closeSavePlanModal();
            if (planRunner) {
                planRunner.refreshPlans();
            }
        } else {
            alert(`Error saving plan: ${result.message}`);
        }
    } catch (err) {
        console.error('Failed to save plan:', err);
        alert('An error occurred while saving the plan.');
    }
}

/**
 * Initialize the application when DOM is loaded.
 */
function init() {
    dashboard = new CIDashboard();
    composer = new PlanComposer();
    planRunner = new PlanRunner();
    
    // Expose instances and functions globally for onclick handlers
    window.dashboard = dashboard;
    window.composer = composer;
    window.planRunner = planRunner;
    
    window.switchTab = switchTab;
    window.startRun = startRun;
    window.cancelRun = cancelRun;
    window.clearDashboard = clearDashboard;
    window.selectAllTargets = selectAllTargets;
    window.clearSelection = clearSelection;
    window.runCustomPlan = runCustomPlan;
    window.savePlan = savePlan;
    window.closeSavePlanModal = closeSavePlanModal;
    window.confirmSavePlan = confirmSavePlan;
}

// Initialize on DOM load
document.addEventListener('DOMContentLoaded', init);
