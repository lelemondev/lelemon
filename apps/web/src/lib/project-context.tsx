'use client';

import { createContext, useContext, useState, useEffect, ReactNode, useCallback, useRef, useMemo } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuth } from '@/lib/auth-context';
import { dashboardAPI, Project as APIProject } from '@/lib/api';

export interface Project {
  id: string;
  name: string;
  apiKey?: string;
  apiKeyHint?: string;
  createdAt: string;
}

interface ProjectContextType {
  projects: Project[];
  currentProject: Project | null;
  setCurrentProject: (project: Project) => void;
  isLoading: boolean;
  refreshProjects: () => Promise<void>;
}

const ProjectContext = createContext<ProjectContextType | undefined>(undefined);

function normalizeProject(p: APIProject): Project {
  return {
    id: p.id,
    name: p.name,
    apiKey: p.apiKey,
    apiKeyHint: p.apiKeyHint || (p.apiKey ? p.apiKey.slice(0, 12) + '...' : undefined),
    createdAt: p.createdAt,
  };
}

export function ProjectProvider({ children }: { children: ReactNode }) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [currentProject, setCurrentProjectState] = useState<Project | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  
  // Use refs for values that shouldn't trigger re-fetch
  const pathnameRef = useRef(pathname);
  const routerRef = useRef(router);
  const hasFetchedRef = useRef(false);
  
  // Update refs when values change
  useEffect(() => {
    pathnameRef.current = pathname;
    routerRef.current = router;
  }, [pathname, router]);

  const fetchProjects = useCallback(async () => {
    if (!isAuthenticated) {
      setProjects([]);
      setCurrentProjectState(null);
      setIsLoading(false);
      return;
    }

    try {
      let data = await dashboardAPI.listProjects();

      // Auto-create project for new users
      if (data.length === 0) {
        const newProject = await dashboardAPI.createProject('My Project');
        data = [newProject];

        // Redirect to config with API key for onboarding (use ref to avoid dependency)
        if (pathnameRef.current !== '/dashboard/config' && newProject.apiKey) {
          routerRef.current.push(`/dashboard/config?welcome=true&key=${encodeURIComponent(newProject.apiKey)}`);
        }
      }

      const normalizedProjects = data.map(normalizeProject);
      setProjects(normalizedProjects);

      // Restore last selected project from localStorage or select first
      const savedProjectId = localStorage.getItem('lelemon_current_project');
      const savedProject = normalizedProjects.find((p) => p.id === savedProjectId);

      if (savedProject) {
        setCurrentProjectState(savedProject);
      } else if (normalizedProjects.length > 0) {
        setCurrentProjectState(normalizedProjects[0]);
        localStorage.setItem('lelemon_current_project', normalizedProjects[0].id);
      }
    } catch (error) {
      console.error('Failed to fetch projects:', error);
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]); // Only depend on isAuthenticated

  const setCurrentProject = useCallback((project: Project) => {
    setCurrentProjectState(project);
    localStorage.setItem('lelemon_current_project', project.id);
  }, []);

  // Only fetch once when auth is ready
  useEffect(() => {
    if (!authLoading && !hasFetchedRef.current) {
      hasFetchedRef.current = true;
      fetchProjects();
    }
  }, [authLoading, fetchProjects]);

  // Reset fetch flag when auth changes (e.g., logout then login)
  useEffect(() => {
    if (!isAuthenticated) {
      hasFetchedRef.current = false;
    }
  }, [isAuthenticated]);

  // Stabilize context value to prevent unnecessary re-renders
  const contextValue = useMemo(
    () => ({
      projects,
      currentProject,
      setCurrentProject,
      isLoading: isLoading || authLoading,
      refreshProjects: fetchProjects,
    }),
    [projects, currentProject, setCurrentProject, isLoading, authLoading, fetchProjects]
  );

  return (
    <ProjectContext.Provider value={contextValue}>
      {children}
    </ProjectContext.Provider>
  );
}

export function useProject() {
  const context = useContext(ProjectContext);
  if (context === undefined) {
    throw new Error('useProject must be used within a ProjectProvider');
  }
  return context;
}
