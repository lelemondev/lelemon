'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { createClient } from '@/lib/supabase/client';

export interface Project {
  id: string;
  name: string;
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
        const data = await response.json();
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
