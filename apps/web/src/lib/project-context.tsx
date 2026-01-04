'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { createClient } from '@/lib/supabase/client';

export interface Project {
  id: string;
  name: string;
  apiKey?: string; // Only present on creation
  apiKeyHint: string | null;
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

export function ProjectProvider({ children }: { children: ReactNode }) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [currentProject, setCurrentProjectState] = useState<Project | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  const fetchProjects = async () => {
    try {
      const supabase = createClient();
      const { data: { user } } = await supabase.auth.getUser();

      if (!user?.email) {
        setProjects([]);
        setCurrentProjectState(null);
        setIsLoading(false);
        return;
      }

      const response = await fetch('/api/v1/projects');
      if (response.ok) {
        let data = await response.json();

        // Auto-create project for new users (email/password login case)
        if (data.length === 0) {
          const createResponse = await fetch('/api/v1/projects', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: 'My Project' }),
          });

          if (createResponse.ok) {
            const newProject = await createResponse.json();
            data = [{
              id: newProject.id,
              name: newProject.name,
              apiKeyHint: newProject.apiKey?.slice(0, 12) + '...',
              createdAt: newProject.createdAt,
            }];

            // Redirect to config with API key for onboarding
            if (pathname !== '/dashboard/config') {
              router.push(`/dashboard/config?welcome=true&key=${encodeURIComponent(newProject.apiKey)}`);
            }
          }
        }

        setProjects(data);

        // Restore last selected project from localStorage or select first
        const savedProjectId = localStorage.getItem('lelemon_current_project');
        const savedProject = data.find((p: Project) => p.id === savedProjectId);

        if (savedProject) {
          setCurrentProjectState(savedProject);
        } else if (data.length > 0) {
          setCurrentProjectState(data[0]);
          localStorage.setItem('lelemon_current_project', data[0].id);
        }
      }
    } catch (error) {
      console.error('Failed to fetch projects:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const setCurrentProject = (project: Project) => {
    setCurrentProjectState(project);
    localStorage.setItem('lelemon_current_project', project.id);
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  return (
    <ProjectContext.Provider
      value={{
        projects,
        currentProject,
        setCurrentProject,
        isLoading,
        refreshProjects: fetchProjects,
      }}
    >
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
